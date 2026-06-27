import { describe, it, expect } from "vitest";
import {
  parseGatewayErrorMessagesToRows,
  serializeGatewayErrorMessageRowsToJSON,
  validateGatewayErrorMessageRows,
  isValidGatewayErrorCode,
} from "../gatewayErrorMessages";

describe("gateway error messages 行编解码", () => {
  it("解析: JSON 对象 → code/message 行", () => {
    const rows = parseGatewayErrorMessagesToRows(
      '{"429":"Please retry later","502":"Upstream unavailable"}',
    );
    expect(rows).toEqual([
      { code: "429", message: "Please retry later" },
      { code: "502", message: "Upstream unavailable" },
    ]);
  });

  it("解析: 空 / 非法 / 数组 / 非字符串值 → 空数组", () => {
    expect(parseGatewayErrorMessagesToRows("")).toEqual([]);
    expect(parseGatewayErrorMessagesToRows("   ")).toEqual([]);
    expect(parseGatewayErrorMessagesToRows("not json")).toEqual([]);
    expect(parseGatewayErrorMessagesToRows('["nope"]')).toEqual([]);
    expect(parseGatewayErrorMessagesToRows('{"429":123}')).toEqual([]);
  });

  it("序列化: 行 → JSON 对象, 去空白行并 trim", () => {
    const json = serializeGatewayErrorMessageRowsToJSON([
      { code: " 429 ", message: " Too many requests " },
      { code: "", message: "" },
      { code: "502", message: "Upstream unavailable" },
    ]);
    expect(JSON.parse(json)).toEqual({
      "429": "Too many requests",
      "502": "Upstream unavailable",
    });
  });

  it("序列化: 空数组 → '{}'", () => {
    expect(serializeGatewayErrorMessageRowsToJSON([])).toBe("{}");
  });

  it("校验: 合法状态码", () => {
    expect(isValidGatewayErrorCode("429")).toBe(true);
    expect(isValidGatewayErrorCode("100")).toBe(true);
    expect(isValidGatewayErrorCode("599")).toBe(true);
    expect(isValidGatewayErrorCode("099")).toBe(false);
    expect(isValidGatewayErrorCode("600")).toBe(false);
    expect(isValidGatewayErrorCode("42")).toBe(false);
    expect(isValidGatewayErrorCode("abc")).toBe(false);
  });

  it("校验: 空行被忽略, 非法/缺失/重复被标记", () => {
    const errors = validateGatewayErrorMessageRows([
      { code: "", message: "" }, // ignored
      { code: "429", message: "ok" }, // valid
      { code: "12", message: "bad code" }, // invalid
      { code: "", message: "no code" }, // code-empty
      { code: "502", message: "" }, // message-empty
      { code: "429", message: "dup" }, // duplicate
    ]);
    expect(errors.get(0)).toBeUndefined();
    expect(errors.get(1)).toBeUndefined();
    expect(errors.get(2)).toBe("code-invalid");
    expect(errors.get(3)).toBe("code-empty");
    expect(errors.get(4)).toBe("message-empty");
    expect(errors.get(5)).toBe("code-duplicate");
  });
});
