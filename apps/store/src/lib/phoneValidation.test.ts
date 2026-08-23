import { describe, expect, it } from "vitest";
import {
  isValidPhone,
  PHONE_RE,
  phoneDigits,
  toFullPhone,
} from "./phoneValidation";

describe("isValidPhone (KAN-53: strict +373 + 8 digits)", () => {
  it("accepts a valid Moldova/PMR number", () => {
    expect(isValidPhone("+37360000000")).toBe(true);
  });

  it("accepts both manager examples", () => {
    expect(isValidPhone("+37360000000")).toBe(true); // ПМР
    expect(isValidPhone("+37368192547")).toBe(true); // МД
  });

  it("trims leading/trailing whitespace before testing", () => {
    expect(isValidPhone("  +37360000000  ")).toBe(true);
  });

  it("rejects empty string", () => {
    expect(isValidPhone("")).toBe(false);
  });

  it("rejects whitespace-only string", () => {
    expect(isValidPhone("   ")).toBe(false);
  });

  it("rejects a number without the +373 prefix", () => {
    expect(isValidPhone("60000000")).toBe(false);
  });

  it("rejects a number with 373 but no +", () => {
    expect(isValidPhone("37360000000")).toBe(false);
  });

  it("rejects 7 digits after +373 (too few)", () => {
    expect(isValidPhone("+3736000000")).toBe(false);
  });

  it("rejects 9 digits after +373 (too many — manager '9 digits' is a typo)", () => {
    expect(isValidPhone("+373600000001")).toBe(false);
  });

  it("rejects non-+373 country codes", () => {
    expect(isValidPhone("+76000000034")).toBe(false); // RUS
    expect(isValidPhone("+38068192547")).toBe(false); // UA
  });

  it("rejects letters mixed in", () => {
    expect(isValidPhone("+3737754abcd")).toBe(false);
  });

  it("rejects spaces inside the number", () => {
    expect(isValidPhone("+373 7754 0982")).toBe(false);
  });

  it("rejects dashes inside the number", () => {
    expect(isValidPhone("+373-7754-0982")).toBe(false);
  });

  it("rejects parentheses inside the number", () => {
    expect(isValidPhone("+373(7754)0982")).toBe(false);
  });
});

describe("PHONE_RE", () => {
  it("is the canonical regex ^\\+373\\d{8}$", () => {
    // Pin the pattern so a future edit to phoneValidation.ts that
    // accidentally changes the regex is caught. Mirror of
    // backend/api/internal/handler/customer.go::phoneRE.
    expect(PHONE_RE.source).toBe("^\\+373\\d{8}$");
  });
});

describe("phoneDigits", () => {
  it("strips the +373 prefix and returns subscriber digits", () => {
    expect(phoneDigits("+37360000000")).toBe("60000000");
  });

  it("strips a bare 373 prefix (no plus)", () => {
    expect(phoneDigits("37360000000")).toBe("60000000");
  });

  it("returns bare subscriber digits unchanged", () => {
    expect(phoneDigits("60000000")).toBe("60000000");
  });

  it("caps at 8 digits", () => {
    expect(phoneDigits("+37360000000999")).toBe("60000000");
  });

  it("tolerates spaces and dashes", () => {
    expect(phoneDigits("+373 6000-0000")).toBe("60000000");
  });

  it("returns empty for empty input", () => {
    expect(phoneDigits("")).toBe("");
  });
});

describe("toFullPhone", () => {
  it("composes the canonical full phone", () => {
    expect(toFullPhone("60000000")).toBe("+37360000000");
  });

  it("returns empty string for empty digits", () => {
    expect(toFullPhone("")).toBe("");
  });
});
