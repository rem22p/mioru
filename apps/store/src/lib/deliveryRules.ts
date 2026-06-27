// apps/store/src/lib/deliveryRules.ts
//
// Single source of truth for "can this delivery method be selected
// for this city?" across the cart checkout (CheckoutPage) and the
// individual-order form (CustomOrderPage). The same rules also
// power the per-method disable styling and the auto-reset useEffect
// that clears a now-blocked selection when the user changes city.
//
// The rules intentionally mirror the backend checks in
// handler/customer.go::CreateOrder so a frontend-passing request
// can't 400 on a city/method mismatch — but the backend is the
// last word, this is just UX.

// Cities in Transnistria (Pridnestrovian Moldavian Republic) —
// physical delivery / pickup is feasible here, but Moldova Post
// doesn't operate because Transnistria has its own postal service
// (ГУП «Почта Приднестровья»). Bus routes also run between PMR
// cities. Spellings are stored in their ё-stripped form; the
// canonical spelling (with ё) is preserved above as a comment for
// editors — runtime comparison goes through `normaliseCity` below.
const PNR_CITIES = new Set([
  "тирасполь", "бендеры", "дубоссары", "рыбница", "григориополь",
  "днестровск", "каменка", "слободзея", "парканы", "ближний хутор",
  "красное", "новотираспольский", "терновка", "маяк", "суклея",
].map((s) => s.replace(/ё/g, "е")));

// Subset of PNR_CITIES where the seller personally hands the
// order over to the customer (or the courier drops it at an
// address). The other PMR cities are still served by bus, but
// require the customer to meet at a fixed stop rather than at
// their door.
const TIRASPOL_BENDERY = new Set(["тирасполь", "бендеры"].map((s) => s.replace(/ё/g, "е")));


// Non-PMR cities reachable by intercity bus from Tiraspol.
const BUS_NON_PNR = new Set(["кишинев", "комрат", "чадыр-лунга"]);

// normaliseCity trims whitespace, lower-cases, and strips ё→е so
// that paste / IME / autofill variants of any Cyrillic city match
// the canonical lookup sets above. Same-shape helper lives on the
// backend in handler/customer.go::CreateOrder — keep them in sync.
function normaliseCity(raw: string): string {
  return raw.trim().toLowerCase().replace(/ё/g, "е");
}

export const DELIVERY_METHODS = [
  "personal",
  "address",
  "bus",
  "express",
  "moldovaPost",
] as const;

export type DeliveryKey = (typeof DELIVERY_METHODS)[number];

// isDeliveryBlocked returns true when the given delivery method
// cannot be selected for the customer's city. The function is
// total: an unknown method (e.g. an old persisted value) returns
// false so a stale form can still submit, and the server will
// return a proper 400.
export function isDeliveryBlocked(method: string, city: string): boolean {
  if (!city) return true; // nothing selectable until city is picked
  const lower = normaliseCity(city);
  const isPnr = PNR_CITIES.has(lower);
  const isTb = TIRASPOL_BENDERY.has(lower);
  switch (method) {
    case "personal":
    case "address":
      return !isTb;
    case "bus":
      // Bus runs between PMR cities, Chisinau, Comrat, Ceadir-Lunga.
      return !(isPnr || BUS_NON_PNR.has(lower));
    case "express":
      // Express post is available across PMR, but the courier
      // doesn't go door-to-door in Tiraspol/Bendery — for those
      // cities the customer is expected to use personal/address
      // pickup instead.
      return !(isPnr && !isTb);
    case "moldovaPost":
      // Moldova Post is the *Republic of Moldova* postal service.
      // Transnistria has its own (ГУП «Почта Приднестровья»),
      // so we hide this option for PMR cities and tell the
      // customer to pick express or bus.
      return isPnr;
    default:
      return false;
  }
}
