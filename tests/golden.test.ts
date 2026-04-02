/**
 * Runtime tests for golden file schemas.
 *
 * Dynamically imports schemas from golden files and tests them against
 * the cases defined in cases.ts. Run inside Docker via docker-typecheck.sh.
 *
 * The ZOD_VERSION env var ("v3" or "v4") determines which zod version is active.
 * Golden files with a @zod-version metadata that doesn't match are skipped.
 */
import { describe, expect, it } from "vitest";
import { readFileSync } from "fs";
import { cases } from "./cases";

// Golden files are copied to /test/golden/ as .ts files by the docker script.
const GOLDEN_DIR = "/test/golden";

// Which zod version we're testing under (set by docker script)
const currentZodVersion = process.env.ZOD_VERSION || "v4";

// Cache for imported golden modules
const moduleCache = new Map<string, Record<string, unknown>>();

// Cache for golden file zod version metadata
const versionCache = new Map<string, string | null>();

function getGoldenZodVersion(golden: string): string | null {
  if (!versionCache.has(golden)) {
    const tsName = golden.replace(/\//g, "__").replace(/\.golden$/, ".ts");
    try {
      const content = readFileSync(`${GOLDEN_DIR}/${tsName}`, "utf-8");
      // The docker script strips // @ comments but the version is in the original.
      // Since prepare_ts strips metadata lines, we check the golden source directly.
      // Actually, the golden source is at /golden/ (mounted read-only from testdata/).
      const goldenSource = readFileSync(
        `/golden/${golden}`,
        "utf-8"
      );
      const match = goldenSource.match(/^\/\/ @zod-version: (v\d+)/m);
      versionCache.set(golden, match ? match[1] : null);
    } catch {
      versionCache.set(golden, null);
    }
  }
  return versionCache.get(golden)!;
}

function shouldSkip(golden: string): boolean {
  const version = getGoldenZodVersion(golden);
  // null means "both versions" — always run
  if (version === null) return false;
  // Skip if the golden file's version doesn't match the current zod version
  return version !== currentZodVersion;
}

async function getSchema(golden: string, schemaName: string) {
  if (!moduleCache.has(golden)) {
    const tsName = golden.replace(/\//g, "__").replace(/\.golden$/, ".ts");
    const mod = await import(`${GOLDEN_DIR}/${tsName}`);
    moduleCache.set(golden, mod);
  }
  const mod = moduleCache.get(golden)!;
  const schema = mod[schemaName];
  if (!schema || typeof (schema as any).safeParse !== "function") {
    throw new Error(
      `Schema "${schemaName}" not found or not a Zod schema in ${golden}`
    );
  }
  return schema as { safeParse: (input: unknown) => any };
}

describe(`Golden file runtime tests (zod@${currentZodVersion})`, () => {
  for (const tc of cases) {
    const skip = shouldSkip(tc.golden);

    const testFn = skip ? it.skip : it;

    testFn(tc.name, async () => {
      const schema = await getSchema(tc.golden, tc.schema);
      const result = schema.safeParse(tc.input);

      if (tc.success) {
        expect(result.success).toBe(true);
        const expected = tc.output !== undefined ? tc.output : tc.input;
        expect(result.data).toEqual(expected);
      } else {
        expect(result.success).toBe(false);
      }
    });
  }
});
