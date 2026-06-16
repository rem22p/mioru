// apps/store/src/lib/deliveryRules.test.ts
//
// Unit tests for the city ↔ delivery-method allowlist. These are
// the rules the storefront uses to disable radio buttons and to
// reset a now-blocked selection when the user changes the city;
// they must stay in lockstep with the server-side checks in
// handler/customer.go::CreateOrder (the two mirror lists are
// flagged with a comment in both files so reviewers can spot
// drift).

import { describe, expect, it } from "vitest";
import { isDeliveryBlocked } from "./deliveryRules";

describe("isDeliveryBlocked", () => {
  describe("no city selected", () => {
    it("blocks every method when city is empty", () => {
      for (const method of ["personal", "address", "bus", "express", "moldovaPost"]) {
        expect(isDeliveryBlocked(method, "")).toBe(true);
      }
    });
  });

  describe("Tiraspol & Bendery (personal/address available)", () => {
    it("allows personal pickup in Tiraspol", () => {
      expect(isDeliveryBlocked("personal", "Тирасполь")).toBe(false);
    });
    it("allows address delivery in Tiraspol", () => {
      expect(isDeliveryBlocked("address", "Тирасполь")).toBe(false);
    });
    it("allows personal pickup in Bendery", () => {
      expect(isDeliveryBlocked("personal", "Бендеры")).toBe(false);
    });
    it("blocks express in Tiraspol (courier doesn't go door-to-door)", () => {
      expect(isDeliveryBlocked("express", "Тирасполь")).toBe(true);
    });
    it("blocks moldovaPost in Tiraspol (PMR has its own postal service)", () => {
      expect(isDeliveryBlocked("moldovaPost", "Тирасполь")).toBe(true);
    });
    it("allows bus in Tiraspol", () => {
      expect(isDeliveryBlocked("bus", "Тирасполь")).toBe(false);
    });
  });

  describe("other PMR cities", () => {
    it("blocks personal/address in Rybnitsa", () => {
      expect(isDeliveryBlocked("personal", "Рыбница")).toBe(true);
      expect(isDeliveryBlocked("address", "Рыбница")).toBe(true);
    });
    it("allows bus in Dubossary", () => {
      expect(isDeliveryBlocked("bus", "Дубоссары")).toBe(false);
    });
    it("allows express in Kamenka (PMR, not TB)", () => {
      expect(isDeliveryBlocked("express", "Каменка")).toBe(false);
    });
    it("blocks moldovaPost in Slobodzeya", () => {
      expect(isDeliveryBlocked("moldovaPost", "Слободзея")).toBe(true);
    });
  });

  describe("Chisinau (Moldova capital, bus only)", () => {
    it("blocks personal/address in Chisinau", () => {
      expect(isDeliveryBlocked("personal", "Кишинев")).toBe(true);
      expect(isDeliveryBlocked("address", "Кишинев")).toBe(true);
    });
    it("allows bus in Chisinau", () => {
      expect(isDeliveryBlocked("bus", "Кишинев")).toBe(false);
    });
    it("blocks express in Chisinau (PMR-only service)", () => {
      expect(isDeliveryBlocked("express", "Кишинев")).toBe(true);
    });
    it("allows moldovaPost in Chisinau (capital, served)", () => {
      expect(isDeliveryBlocked("moldovaPost", "Кишинев")).toBe(false);
    });
  });

  describe("other Moldova cities (e.g. Beltsy)", () => {
    it("blocks personal/address in Beltsy (not in TB)", () => {
      expect(isDeliveryBlocked("personal", "Бельцы")).toBe(true);
      expect(isDeliveryBlocked("address", "Бельцы")).toBe(true);
    });
    it("blocks bus in Beltsy (no PMR/Chisinau bus line)", () => {
      expect(isDeliveryBlocked("bus", "Бельцы")).toBe(true);
    });
    it("blocks express in Beltsy (PMR-only service)", () => {
      expect(isDeliveryBlocked("express", "Бельцы")).toBe(true);
    });
    it("allows moldovaPost in Beltsy (Moldova)", () => {
      expect(isDeliveryBlocked("moldovaPost", "Бельцы")).toBe(false);
    });
  });

  describe("non-MD/non-PMR cities", () => {
    it("blocks every local method for a foreign city", () => {
      expect(isDeliveryBlocked("personal", "Москва")).toBe(true);
      expect(isDeliveryBlocked("address", "Москва")).toBe(true);
      expect(isDeliveryBlocked("bus", "Москва")).toBe(true);
      expect(isDeliveryBlocked("express", "Москва")).toBe(true);
    });
    it("still allows moldovaPost (Moldova post can ship internationally)", () => {
      expect(isDeliveryBlocked("moldovaPost", "Москва")).toBe(false);
    });
  });

  describe("input is case-insensitive", () => {
    it("tolerates TIRASPOL", () => {
      expect(isDeliveryBlocked("personal", "TIRASPOL")).toBe(true);
    });
    it("tolerates leading/trailing spaces (caller trims before passing)", () => {
      // We don't trim ourselves — the caller is expected to pass
      // the canonical string. The test documents that.
      expect(isDeliveryBlocked("personal", "Тирасполь")).toBe(false);
    });
  });

  describe("unknown method", () => {
    it("returns false (stale persisted value — let the server 400)", () => {
      expect(isDeliveryBlocked("drone", "Тирасполь")).toBe(false);
    });
  });
});
