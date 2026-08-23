import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { useState } from "react";
import PhoneInput from "./PhoneInput";
import "@testing-library/jest-dom/vitest";

function Harness({ initial = "" }: { initial?: string }) {
  const [val, setVal] = useState(initial);
  return (
    <div>
      <PhoneInput value={val} onChange={setVal} data-testid="phone" />
      <output data-testid="full">{val}</output>
    </div>
  );
}

describe("PhoneInput (KAN-53)", () => {
  it("always shows the +373 prefix", () => {
    render(<Harness />);
    expect(screen.getByText("+373")).toBeInTheDocument();
  });

  it("emits canonical +373XXXXXXXX as digits are typed", () => {
    render(<Harness />);
    fireEvent.change(screen.getByTestId("phone"), {
      target: { value: "60000000" },
    });
    expect(screen.getByTestId("full")).toHaveTextContent("+37360000000");
  });

  it("caps input at 8 digits — nothing more can be entered", () => {
    render(<Harness />);
    // The DOM is the first cap: the browser refuses the 9th character on
    // typing, pasting and Playwright's fill() alike.
    expect(screen.getByTestId("phone")).toHaveAttribute("maxlength", "8");
    // The parser is the second: a stored full number longer than +373 + 8
    // digits keeps only the subscriber part.
    fireEvent.change(screen.getByTestId("phone"), {
      target: { value: "+37360000000999" },
    });
    expect(screen.getByTestId("full")).toHaveTextContent("+37360000000");
    expect(screen.getByTestId("phone")).toHaveValue("60000000");
  });

  it("does not turn a foreign number into a +373 one", () => {
    // A legacy "+79161234567" (valid before KAN-53) must not surface as the
    // plausible-but-different "+37379161234" — the field stays empty instead.
    render(<Harness initial="+79161234567" />);
    expect(screen.getByTestId("phone")).toHaveValue("");
    // The stored value is not silently wiped either: it survives untouched
    // until the user types, and the server rejects it loudly on save
    // (customer.go phoneRE on both Register/UpdateProfile and CreateOrder).
    expect(screen.getByTestId("full")).toHaveTextContent("+79161234567");
  });

  it("strips non-digits (letters, spaces, dashes)", () => {
    render(<Harness />);
    fireEvent.change(screen.getByTestId("phone"), {
      target: { value: "60-00 ab0000" },
    });
    expect(screen.getByTestId("full")).toHaveTextContent("+37360000000");
  });

  it("parses an existing full value back into subscriber digits", () => {
    render(<Harness initial="+37368192547" />);
    expect(screen.getByTestId("phone")).toHaveValue("68192547");
  });

  it("emits empty string when cleared", () => {
    render(<Harness initial="+37360000000" />);
    fireEvent.change(screen.getByTestId("phone"), { target: { value: "" } });
    expect(screen.getByTestId("full")).toHaveTextContent("");
  });

  it("keeps the digits-only value while the parent holds the full phone", () => {
    render(<Harness />);
    fireEvent.change(screen.getByTestId("phone"), {
      target: { value: "60000000" },
    });
    expect(screen.getByTestId("phone")).toHaveValue("60000000");
    expect(screen.getByTestId("full")).toHaveTextContent("+37360000000");
  });
});
