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
import { existsSync, readFileSync } from "fs";
import { cases } from "./cases";

// Golden files are copied to /test/golden/ as .ts files by the docker script.
const GOLDEN_DIR = "/test/golden";

// Which zod version we're testing under (set by docker script)
const currentZodVersion = process.env.ZOD_VERSION || "v4";

// Cache for imported golden modules
const moduleCache = new Map<string, Record<string, unknown>>();

// Cache for golden file zod version metadata
const versionCache = new Map<string, string | null>();

/**
 * Resolves a golden path to a concrete .golden file path.
 *
 * If the path ends with ".golden", it is used as-is.
 * Otherwise it is treated as a directory containing v3.golden / v4.golden,
 * and the file matching currentZodVersion is returned.
 */
function resolveGolden(golden: string): string {
  if (golden.endsWith(".golden")) {
    return golden;
  }
  // Directory path — check that at least one version file exists
  const dir = golden.endsWith("/") ? golden : golden + "/";
  const hasV3 = existsSync(`/golden/${dir}v3.golden`);
  const hasV4 = existsSync(`/golden/${dir}v4.golden`);
  if (!hasV3 && !hasV4) {
    throw new Error(
      `No golden files found in directory "${golden}" — expected v3.golden or v4.golden`
    );
  }
  return dir + currentZodVersion + ".golden";
}

function getGoldenZodVersion(golden: string): string | null {
  const resolved = resolveGolden(golden);
  if (!versionCache.has(resolved)) {
    try {
      const goldenSource = readFileSync(`/golden/${resolved}`, "utf-8");
      const match = goldenSource.match(/^\/\/ @zod-version: (v\d+)/m);
      versionCache.set(resolved, match ? match[1] : null);
    } catch {
      versionCache.set(resolved, null);
    }
  }
  return versionCache.get(resolved)!;
}

function shouldSkip(golden: string): boolean {
  const version = getGoldenZodVersion(golden);
  // null means "both versions" — always run
  if (version === null) return false;
  // Skip if the golden file's version doesn't match the current zod version
  return version !== currentZodVersion;
}

async function getSchema(golden: string, schemaName: string) {
  const resolved = resolveGolden(golden);
  if (!moduleCache.has(resolved)) {
    const tsName = resolved.replace(/\//g, "__").replace(/\.golden$/, ".ts");
    const mod = await import(`${GOLDEN_DIR}/${tsName}`);
    moduleCache.set(resolved, mod);
  }
  const mod = moduleCache.get(resolved)!;
  const schema = mod[schemaName];
  if (!schema || typeof (schema as any).safeParse !== "function") {
    throw new Error(
      `Schema "${schemaName}" not found or not a Zod schema in ${resolved}`
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
