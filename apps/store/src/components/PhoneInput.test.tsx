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
    fireEvent.change(screen.getByTestId("phone"), {
      target: { value: "600000001234" },
    });
    // Only the first 8 digits survive
    expect(screen.getByTestId("full")).toHaveTextContent("+37360000000");
    expect(screen.getByTestId("phone")).toHaveValue("60000000");
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
