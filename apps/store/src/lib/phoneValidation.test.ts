import { describe, expect, it } from "vitest";
import { isValidPhone, PHONE_RE } from "./phoneValidation";

describe("isValidPhone", () => {
  it("accepts a valid +<country><digits> phone", () => {
    expect(isValidPhone("+37370001234")).toBe(true);
  });

  it("accepts a phone without the leading +", () => {
    expect(isValidPhone("37370001234")).toBe(true);
  });

  it("accepts the minimum 7-digit length", () => {
    expect(isValidPhone("+1234567")).toBe(true);
  });

  it("accepts the maximum 15-digit length", () => {
    expect(isValidPhone("+123456789012345")).toBe(true);
  });

  it("trims leading/trailing whitespace before testing", () => {
    expect(isValidPhone("  +37370001234  ")).toBe(true);
  });

  it("rejects empty string", () => {
    expect(isValidPhone("")).toBe(false);
  });

  it("rejects whitespace-only string", () => {
    expect(isValidPhone("   ")).toBe(false);
  });

  it("rejects 6 digits (below minimum)", () => {
    expect(isValidPhone("+123456")).toBe(false);
  });

  it("rejects 16 digits (above maximum)", () => {
    expect(isValidPhone("+1234567890123456")).toBe(false);
  });

  it("rejects letters mixed in", () => {
    expect(isValidPhone("+3737000abcd")).toBe(false);
  });

  it("rejects spaces inside the number", () => {
    expect(isValidPhone("+373 700 012 34")).toBe(false);
  });

  it("rejects dashes inside the number", () => {
    expect(isValidPhone("+373-700-012-34")).toBe(false);
  });

  it("rejects parentheses inside the number", () => {
    expect(isValidPhone("+373(700)012-34")).toBe(false);
  });
});

describe("PHONE_RE", () => {
  it("is the canonical regex ^\\+?\\d{7,15}$", () => {
    // Pin the pattern so a future edit to phoneValidation.ts that
    // accidentally changes the regex is caught. Mirror of
    // backend/api/internal/handler/customer.go::phoneRE.
    expect(PHONE_RE.source).toBe("^\\+?\\d{7,15}$");
  });
});