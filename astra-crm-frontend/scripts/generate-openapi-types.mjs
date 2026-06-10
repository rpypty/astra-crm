import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { parseDocument } from "yaml";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const projectRoot = path.resolve(__dirname, "..");
const specPath = path.resolve(projectRoot, "../docs/openapi.yaml");
const outputPath = path.resolve(projectRoot, "src/lib/generated/openapi.ts");
const checkOnly = process.argv.includes("--check");

const source = fs.readFileSync(specPath, "utf8");
const document = parseDocument(source, { prettyErrors: true });
if (document.errors.length > 0) {
  throw new Error(document.errors.map((error) => error.message).join("\n"));
}

const spec = document.toJSON();
validateSpec(spec);

const generated = renderTypes(spec);
if (checkOnly) {
  const current = fs.existsSync(outputPath) ? fs.readFileSync(outputPath, "utf8") : "";
  if (current !== generated) {
    throw new Error("Generated OpenAPI types are stale. Run npm run openapi:generate.");
  }
} else {
  fs.mkdirSync(path.dirname(outputPath), { recursive: true });
  fs.writeFileSync(outputPath, generated);
}

function validateSpec(spec) {
  if (!String(spec.openapi ?? "").startsWith("3.")) {
    throw new Error("docs/openapi.yaml must be an OpenAPI 3.x document.");
  }

  const sessionDescription = spec.components?.securitySchemes?.sessionCookie?.description ?? "";
  for (const requiredText of ["httpOnly", "SameSite=Lax", "Secure"]) {
    if (!sessionDescription.includes(requiredText)) {
      throw new Error(`sessionCookie description must document ${requiredText}.`);
    }
  }

  const errorSchema = spec.components?.schemas?.ErrorResponse;
  if (!errorSchema?.properties?.error?.properties?.code || !errorSchema?.properties?.error?.properties?.message) {
    throw new Error("ErrorResponse must contain error.code and error.message.");
  }

  for (const [route, pathItem] of Object.entries(spec.paths ?? {})) {
    for (const method of ["get", "post", "patch", "delete", "put"]) {
      const operation = pathItem?.[method];
      if (!operation) {
        continue;
      }
      const defaultResponse = operation.responses?.default;
      if (defaultResponse?.$ref !== "#/components/responses/Error") {
        throw new Error(`${method.toUpperCase()} ${route} must define default Error response.`);
      }
    }
  }
}

function renderTypes(spec) {
  const schemas = spec.components?.schemas ?? {};
  const renderedSchemas = Object.entries(schemas)
    .map(([name, schema]) => `    ${safeKey(name)}: ${schemaToTs(schema)};`)
    .join("\n");

  return `/* eslint-disable */\n` +
    `/* This file is generated from docs/openapi.yaml. Run npm run openapi:generate. */\n\n` +
    `export type components = {\n` +
    `  schemas: {\n${renderedSchemas}\n  };\n` +
    `};\n\n` +
    `export type ApiSchema<Name extends keyof components["schemas"]> = components["schemas"][Name];\n`;
}

function schemaToTs(schema) {
  if (!schema) {
    return "unknown";
  }
  if (schema.$ref) {
    return refToTs(schema.$ref);
  }
  if (schema.allOf) {
    return schema.allOf.map(schemaToTs).join(" & ");
  }
  if (schema.oneOf || schema.anyOf) {
    return (schema.oneOf ?? schema.anyOf).map(schemaToTs).join(" | ");
  }
  if (Array.isArray(schema.enum)) {
    return schema.enum.map((value) => JSON.stringify(value)).join(" | ");
  }

  const nullable = schema.nullable ? " | null" : "";
  switch (schema.type) {
    case "integer":
    case "number":
      return `number${nullable}`;
    case "string":
      return `string${nullable}`;
    case "boolean":
      return `boolean${nullable}`;
    case "array":
      return `${schemaToTs(schema.items)}[]${nullable}`;
    case "object":
    case undefined:
      return `${objectToTs(schema)}${nullable}`;
    default:
      return `unknown${nullable}`;
  }
}

function objectToTs(schema) {
  const properties = schema.properties ?? {};
  const required = new Set(schema.required ?? []);
  const lines = Object.entries(properties).map(([name, child]) => {
    const optional = required.has(name) ? "" : "?";
    return `  ${safeKey(name)}${optional}: ${schemaToTs(child)};`;
  });

  if (schema.additionalProperties) {
    const valueSchema = schema.additionalProperties === true ? "unknown" : schemaToTs(schema.additionalProperties);
    lines.push(`  [key: string]: ${valueSchema};`);
  }

  if (lines.length === 0) {
    return "Record<string, unknown>";
  }

  return `{\n${lines.join("\n")}\n}`;
}

function refToTs(ref) {
  const prefix = "#/components/schemas/";
  if (!ref.startsWith(prefix)) {
    return "unknown";
  }
  return `components["schemas"][${JSON.stringify(ref.slice(prefix.length))}]`;
}

function safeKey(key) {
  return /^[A-Za-z_$][A-Za-z0-9_$]*$/.test(key) ? key : JSON.stringify(key);
}
