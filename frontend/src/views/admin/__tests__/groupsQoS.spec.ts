import { describe, expect, it } from "vitest";
import {
  createGroupQoSModelMappingRow,
  createGroupQoSTierRow,
  groupQoSTiersToAPI,
  groupQoSTiersToRows,
  hasGroupQoSErrors,
  normalizeGroupQoSMetric,
  normalizeGroupQoSWindow,
  validateGroupQoSTiers,
  type GroupQoSTierRow,
} from "../groupsQoS";
import type { GroupQoSTier } from "@/types";

const tierRow = (overrides: Partial<GroupQoSTierRow> = {}): GroupQoSTierRow => ({
  ...createGroupQoSTierRow(),
  window: "daily",
  thresholdUsd: "50",
  block: true,
  ...overrides,
});

describe("normalizeGroupQoSMetric", () => {
  it("defaults to list cost", () => {
    expect(normalizeGroupQoSMetric(undefined)).toBe("list");
    expect(normalizeGroupQoSMetric("")).toBe("list");
    expect(normalizeGroupQoSMetric("nonsense")).toBe("list");
  });

  it("accepts charged cost", () => {
    expect(normalizeGroupQoSMetric(" CHARGED ")).toBe("charged");
  });
});

describe("normalizeGroupQoSWindow", () => {
  it("accepts the three supported windows", () => {
    expect(normalizeGroupQoSWindow("daily")).toBe("daily");
    expect(normalizeGroupQoSWindow(" WEEKLY ")).toBe("weekly");
    expect(normalizeGroupQoSWindow("monthly")).toBe("monthly");
  });

  it("rejects anything else", () => {
    expect(normalizeGroupQoSWindow("hourly")).toBe("");
    expect(normalizeGroupQoSWindow(undefined)).toBe("");
  });
});

describe("groupQoSTiersToRows / groupQoSTiersToAPI", () => {
  it("round-trips a ladder", () => {
    const tiers: GroupQoSTier[] = [
      {
        window: "daily",
        threshold_usd: 50,
        model_mappings: [{ from: "gpt-5.6-sol*", to: "gpt-5.6-terra" }],
        max_reasoning_effort: "medium",
      },
      { window: "daily", threshold_usd: 200, block: true },
    ];
    expect(groupQoSTiersToAPI(groupQoSTiersToRows(tiers))).toEqual(tiers);
  });

  it("drops half-filled model mappings", () => {
    const rows = [
      tierRow({
        block: false,
        modelMappings: [
          createGroupQoSModelMappingRow({ from: "a", to: "b" }),
          createGroupQoSModelMappingRow({ from: "c", to: "" }),
        ],
      }),
    ];
    expect(groupQoSTiersToAPI(rows)[0].model_mappings).toEqual([
      { from: "a", to: "b" },
    ]);
  });

  it("omits untouched optional actions", () => {
    const [tier] = groupQoSTiersToAPI([tierRow()]);
    expect(tier).toEqual({ window: "daily", threshold_usd: 50, block: true });
  });
});

describe("validateGroupQoSTiers", () => {
  it("accepts a well-formed ascending ladder", () => {
    const rows = [
      tierRow({ thresholdUsd: "50" }),
      tierRow({ thresholdUsd: "100" }),
    ];
    expect(hasGroupQoSErrors(validateGroupQoSTiers(rows, "openai"))).toBe(false);
  });

  it("rejects non-ascending thresholds in the same window", () => {
    const rows = [
      tierRow({ thresholdUsd: "100" }),
      tierRow({ thresholdUsd: "50" }),
    ];
    const errors = validateGroupQoSTiers(rows, "openai");
    expect(errors.tiers[rows[1].id]).toBe("thresholdNotAscending");
  });

  it("allows independent ladders per window", () => {
    const rows = [
      tierRow({ window: "daily", thresholdUsd: "100" }),
      tierRow({ window: "weekly", thresholdUsd: "50" }),
    ];
    expect(hasGroupQoSErrors(validateGroupQoSTiers(rows, "openai"))).toBe(false);
  });

  it("rejects a tier with no action", () => {
    const rows = [tierRow({ block: false })];
    expect(validateGroupQoSTiers(rows, "openai").tiers[rows[0].id]).toBe(
      "noAction",
    );
  });

  it("rejects a missing or negative threshold", () => {
    const empty = tierRow({ thresholdUsd: "" });
    expect(validateGroupQoSTiers([empty], "openai").tiers[empty.id]).toBe(
      "thresholdInvalid",
    );
    const negative = tierRow({ thresholdUsd: "-1" });
    expect(validateGroupQoSTiers([negative], "openai").tiers[negative.id]).toBe(
      "thresholdInvalid",
    );
  });

  it("rejects a non-integer or negative rpm limit", () => {
    const row = tierRow({ block: false, rpmLimit: "2.5" });
    expect(validateGroupQoSTiers([row], "openai").tiers[row.id]).toBe(
      "rpmInvalid",
    );
  });

  it("rejects reasoning effort on a platform that has none", () => {
    const row = tierRow({ block: false, maxReasoningEffort: "low" });
    expect(validateGroupQoSTiers([row], "anthropic").tiers[row.id]).toBe(
      "effortUnsupported",
    );
    expect(
      validateGroupQoSTiers([row], "openai").tiers[row.id],
    ).toBeUndefined();
  });

  it("flags duplicate and wildcard-target mappings", () => {
    const dupA = createGroupQoSModelMappingRow({ from: "sol", to: "luna" });
    const dupB = createGroupQoSModelMappingRow({ from: "SOL", to: "terra" });
    const wild = createGroupQoSModelMappingRow({ from: "x", to: "y*" });
    const row = tierRow({ modelMappings: [dupA, dupB, wild] });
    const errors = validateGroupQoSTiers([row], "openai");
    expect(errors.mappings[dupA.id]?.from).toBe("duplicateFrom");
    expect(errors.mappings[dupB.id]?.from).toBe("duplicateFrom");
    expect(errors.mappings[wild.id]?.to).toBe("wildcardTarget");
  });

  it("flags an incomplete mapping", () => {
    const partial = createGroupQoSModelMappingRow({ from: "", to: "" });
    const row = tierRow({ modelMappings: [partial] });
    const errors = validateGroupQoSTiers([row], "openai");
    expect(errors.mappings[partial.id]?.from).toBe("fromRequired");
    expect(errors.mappings[partial.id]?.to).toBe("toRequired");
  });
});
