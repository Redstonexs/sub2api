import type {
  GroupPlatform,
  GroupQoSMetric,
  GroupQoSModelMapping,
  GroupQoSTier,
  GroupQoSWindow,
} from "@/types";
import { supportsReasoningEffortPolicyPlatform } from "./groupsReasoningEffort";

export const groupQoSWindows = ["daily", "weekly", "monthly"] as const;
export const groupQoSMetrics = ["list", "charged"] as const;

const MAX_QOS_TIERS = 16;
const MAX_QOS_MODEL_MAPPINGS = 64;

export function normalizeGroupQoSMetric(
  value: string | null | undefined,
): GroupQoSMetric {
  // Anything unrecognized falls back to list cost — the safer default, since it
  // counts undiscounted consumption and a deep group discount cannot mask abuse.
  return value?.trim().toLowerCase() === "charged" ? "charged" : "list";
}

export function normalizeGroupQoSWindow(
  value: string | null | undefined,
): GroupQoSWindow | "" {
  const normalized = value?.trim().toLowerCase() ?? "";
  return groupQoSWindows.some((w) => w === normalized)
    ? (normalized as GroupQoSWindow)
    : "";
}

export interface GroupQoSModelMappingRow extends GroupQoSModelMapping {
  id: string;
}

export interface GroupQoSTierRow {
  id: string;
  window: GroupQoSWindow;
  thresholdUsd: string;
  modelMappings: GroupQoSModelMappingRow[];
  maxReasoningEffort: string;
  rpmLimit: string;
  block: boolean;
}

export type GroupQoSTierErrorCode =
  | "windowRequired"
  | "thresholdInvalid"
  | "thresholdNotAscending"
  | "noAction"
  | "tooManyTiers"
  | "tooManyMappings"
  | "rpmInvalid"
  | "effortUnsupported";

export type GroupQoSMappingErrorCode =
  | "fromRequired"
  | "toRequired"
  | "duplicateFrom"
  | "wildcardTarget";

export interface GroupQoSErrors {
  tiers: Record<string, GroupQoSTierErrorCode>;
  mappings: Record<
    string,
    Partial<Record<"from" | "to", GroupQoSMappingErrorCode>>
  >;
}

let nextRowID = 0;

export function createGroupQoSModelMappingRow(
  mapping: Partial<GroupQoSModelMapping> = {},
): GroupQoSModelMappingRow {
  nextRowID += 1;
  return {
    id: `qos-mapping-${nextRowID}`,
    from: mapping.from ?? "",
    to: mapping.to ?? "",
  };
}

export function createGroupQoSTierRow(
  tier: Partial<GroupQoSTier> = {},
): GroupQoSTierRow {
  nextRowID += 1;
  return {
    id: `qos-tier-${nextRowID}`,
    window: normalizeGroupQoSWindow(tier.window) || "daily",
    thresholdUsd:
      tier.threshold_usd === undefined || tier.threshold_usd === null
        ? ""
        : String(tier.threshold_usd),
    modelMappings: (tier.model_mappings ?? []).map(
      createGroupQoSModelMappingRow,
    ),
    maxReasoningEffort: tier.max_reasoning_effort ?? "",
    rpmLimit:
      tier.rpm_limit === undefined || tier.rpm_limit === null
        ? ""
        : String(tier.rpm_limit),
    block: tier.block ?? false,
  };
}

export function groupQoSTiersToRows(
  tiers?: GroupQoSTier[] | null,
): GroupQoSTierRow[] {
  return (tiers ?? []).map(createGroupQoSTierRow);
}

export function groupQoSTiersToAPI(rows: GroupQoSTierRow[]): GroupQoSTier[] {
  return rows.map((row) => {
    const tier: GroupQoSTier = {
      window: row.window,
      threshold_usd: Number(row.thresholdUsd) || 0,
    };
    const mappings = row.modelMappings
      .filter((m) => m.from.trim() && m.to.trim())
      .map((m) => ({ from: m.from.trim(), to: m.to.trim() }));
    if (mappings.length) tier.model_mappings = mappings;
    if (row.maxReasoningEffort.trim()) {
      tier.max_reasoning_effort = row.maxReasoningEffort.trim();
    }
    if (row.rpmLimit.trim()) tier.rpm_limit = Number(row.rpmLimit);
    if (row.block) tier.block = true;
    return tier;
  });
}

function tierHasAction(row: GroupQoSTierRow): boolean {
  return (
    row.block ||
    row.rpmLimit.trim() !== "" ||
    row.maxReasoningEffort.trim() !== "" ||
    row.modelMappings.some((m) => m.from.trim() && m.to.trim())
  );
}

/**
 * Mirrors the backend's NormalizeGroupQoSTiers so an admin sees the problem
 * before saving rather than as a 400 afterwards.
 */
export function validateGroupQoSTiers(
  rows: GroupQoSTierRow[],
  platform: GroupPlatform,
): GroupQoSErrors {
  const errors: GroupQoSErrors = { tiers: {}, mappings: {} };
  const lastThresholdByWindow = new Map<string, number>();

  rows.forEach((row, index) => {
    if (index >= MAX_QOS_TIERS) {
      errors.tiers[row.id] = "tooManyTiers";
      return;
    }
    if (!normalizeGroupQoSWindow(row.window)) {
      errors.tiers[row.id] = "windowRequired";
      return;
    }

    const threshold = Number(row.thresholdUsd);
    if (row.thresholdUsd.trim() === "" || !Number.isFinite(threshold) || threshold < 0) {
      errors.tiers[row.id] = "thresholdInvalid";
    } else {
      const previous = lastThresholdByWindow.get(row.window);
      if (previous !== undefined && threshold <= previous) {
        // Ordering must be unambiguous: the highest matching tier wins.
        errors.tiers[row.id] = "thresholdNotAscending";
      }
      lastThresholdByWindow.set(row.window, threshold);
    }

    if (row.rpmLimit.trim() !== "") {
      const rpm = Number(row.rpmLimit);
      if (!Number.isInteger(rpm) || rpm < 0) {
        errors.tiers[row.id] = "rpmInvalid";
      }
    }

    if (
      row.maxReasoningEffort.trim() !== "" &&
      !supportsReasoningEffortPolicyPlatform(platform)
    ) {
      errors.tiers[row.id] = "effortUnsupported";
    }

    if (!errors.tiers[row.id] && !tierHasAction(row)) {
      errors.tiers[row.id] = "noAction";
    }

    if (row.modelMappings.length > MAX_QOS_MODEL_MAPPINGS) {
      errors.tiers[row.id] = "tooManyMappings";
    }

    const seen = new Map<string, GroupQoSModelMappingRow[]>();
    row.modelMappings.forEach((mapping) => {
      const from = mapping.from.trim();
      const to = mapping.to.trim();
      if (!from) {
        errors.mappings[mapping.id] = {
          ...errors.mappings[mapping.id],
          from: "fromRequired",
        };
      } else {
        const key = from.toLowerCase();
        seen.set(key, [...(seen.get(key) ?? []), mapping]);
      }
      if (!to) {
        errors.mappings[mapping.id] = {
          ...errors.mappings[mapping.id],
          to: "toRequired",
        };
      } else if (to.includes("*")) {
        // A wildcard target would forward a literal "*" upstream.
        errors.mappings[mapping.id] = {
          ...errors.mappings[mapping.id],
          to: "wildcardTarget",
        };
      }
    });
    seen.forEach((duplicates) => {
      if (duplicates.length < 2) return;
      duplicates.forEach((mapping) => {
        errors.mappings[mapping.id] = {
          ...errors.mappings[mapping.id],
          from: "duplicateFrom",
        };
      });
    });
  });

  return errors;
}

export function hasGroupQoSErrors(errors: GroupQoSErrors): boolean {
  return (
    Object.keys(errors.tiers).length > 0 ||
    Object.keys(errors.mappings).length > 0
  );
}
