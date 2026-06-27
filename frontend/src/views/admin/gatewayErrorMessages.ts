// Helpers for the "Gateway Error Messages" settings form.
//
// The backend stores this setting as a JSON object mapping a 3-digit HTTP
// status code to the user-facing message returned by the gateway. The admin UI
// edits it as a list of rows so users never have to hand-write JSON.

export interface GatewayErrorMessageRow {
  code: string;
  message: string;
}

// Common gateway-relevant HTTP status codes offered as datalist suggestions.
// Users may still type any valid 3-digit code (100–599).
export const COMMON_GATEWAY_ERROR_CODES: ReadonlyArray<string> = [
  "400",
  "401",
  "403",
  "404",
  "408",
  "409",
  "413",
  "422",
  "429",
  "500",
  "501",
  "502",
  "503",
  "504",
  "524",
  "529",
];

// Mirrors the backend rule (handler/admin/setting_handler.go): first digit 1-5,
// remaining two digits 0-9 (i.e. 100–599).
const CODE_PATTERN = /^[1-5]\d{2}$/;

export function isValidGatewayErrorCode(code: string): boolean {
  return CODE_PATTERN.test(code.trim());
}

export function parseGatewayErrorMessagesToRows(
  raw: string,
): GatewayErrorMessageRow[] {
  if (!raw || !raw.trim()) return [];
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return [];
  }
  if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
    return [];
  }
  return Object.entries(parsed as Record<string, unknown>)
    .filter(([, value]) => typeof value === "string")
    .map(([code, value]) => ({ code: String(code), message: value as string }));
}

export function serializeGatewayErrorMessageRowsToJSON(
  rows: GatewayErrorMessageRow[],
): string {
  const result: Record<string, string> = {};
  for (const row of rows) {
    const code = row.code.trim();
    const message = row.message.trim();
    // Drop fully-blank rows the user added but never filled in.
    if (!code && !message) continue;
    result[code] = message;
  }
  return JSON.stringify(result);
}

export type GatewayErrorRowErrorKind =
  | "code-empty"
  | "code-invalid"
  | "code-duplicate"
  | "message-empty";

// Validates the rows and returns a map of row index -> first error found.
// Fully-blank rows are ignored (they are dropped on serialize). Code errors
// take precedence over message errors for a given row.
export function validateGatewayErrorMessageRows(
  rows: GatewayErrorMessageRow[],
): Map<number, GatewayErrorRowErrorKind> {
  const errors = new Map<number, GatewayErrorRowErrorKind>();
  const seen = new Set<string>();
  rows.forEach((row, index) => {
    const code = row.code.trim();
    const message = row.message.trim();
    if (!code && !message) return;
    if (!code) {
      errors.set(index, "code-empty");
      return;
    }
    if (!isValidGatewayErrorCode(code)) {
      errors.set(index, "code-invalid");
      return;
    }
    if (seen.has(code)) {
      errors.set(index, "code-duplicate");
      return;
    }
    seen.add(code);
    if (!message) {
      errors.set(index, "message-empty");
    }
  });
  return errors;
}
