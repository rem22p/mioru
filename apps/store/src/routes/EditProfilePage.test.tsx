import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { HelmetProvider } from "@dr.pogodin/react-helmet";
import EditProfilePage from "./EditProfilePage";
import { useAuthStore } from "@/stores/authStore";
import { fetchStoreCustomerUpdate } from "@/lib/api";
import type { User } from "@/types";
import "@testing-library/jest-dom/vitest";

// ── Mocks (non-store infra only) ──────────────────────────────────────────

vi.mock("framer-motion", () => ({
  motion: new Proxy(
    {},
    {
      get: (_t, tag: string) =>
        ({
          h1: (p: React.HTMLAttributes<HTMLHeadingElement>) => <h1 {...p} />,
          form: (p: React.FormHTMLAttributes<HTMLFormElement>) => <form {...p} />,
        })[tag] || ((p: React.HTMLAttributes<HTMLElement>) => <div {...p} />),
    },
  ),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (k: string, opts?: Record<string, string>) =>
      opts?.phone ? `${k}:${opts.phone}` : k,
  }),
}));

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router-dom")>();
  return { ...actual, useNavigate: () => mockNavigate };
});

vi.mock("@/lib/api", () => ({
  fetchStoreCustomerUpdate: vi.fn(() => Promise.resolve({ ok: true })),
}));

// ── Fixtures ──────────────────────────────────────────────────────────────

function makeUser(phone: string): User {
  return {
    id: "1",
    name: "E2E",
    email: "e2e@example.com",
    firstName: "E2E",
    lastName: "",
    phone,
    avatarParams: {} as unknown as User["avatarParams"],
    xpBalance: 0,
    vipLevel: 0,
    telegram: { linked: false },
  };
}

function renderWithPhone(phone: string) {
  useAuthStore.setState({
    isAuthenticated: true,
    user: makeUser(phone),
    loading: false,
    error: null,
  });
  // fetchMe() would hit the network after a successful save.
  useAuthStore.setState({ fetchMe: vi.fn(() => Promise.resolve()) } as never);
  return render(
    <HelmetProvider>
      <MemoryRouter>
        <EditProfilePage />
      </MemoryRouter>
    </HelmetProvider>,
  );
}

async function save() {
  fireEvent.change(screen.getByPlaceholderText("auth.password"), {
    target: { value: "hunter2" },
  });
  fireEvent.submit(screen.getByTestId("profile-save"));
}

describe("EditProfilePage — phone under the +373 contract", () => {
  beforeEach(() => {
    mockNavigate.mockClear();
    vi.mocked(fetchStoreCustomerUpdate).mockClear();
  });

  it("prefills a +373 number and submits it unchanged", async () => {
    renderWithPhone("+37360000000");

    expect(screen.getByTestId("profile-phone")).toHaveValue("60000000");
    expect(screen.queryByTestId("profile-phone-dropped")).not.toBeInTheDocument();

    await save();

    await waitFor(() =>
      expect(fetchStoreCustomerUpdate).toHaveBeenCalledWith(
        expect.objectContaining({ phone: "+37360000000" }),
      ),
    );
  });

  it("does not submit a pre-KAN-53 number that the field cannot show", async () => {
    // The server rejects "+79161234567" since KAN-53. Submitting it anyway
    // would 400 the whole save — the user could no longer edit their name,
    // with the error naming a phone field that looks empty on screen.
    renderWithPhone("+79161234567");

    expect(screen.getByTestId("profile-phone")).toHaveValue("");

    await save();

    await waitFor(() =>
      expect(fetchStoreCustomerUpdate).toHaveBeenCalledWith(
        expect.objectContaining({ phone: "" }),
      ),
    );
  });

  it("says the old number is being dropped instead of wiping it silently", () => {
    renderWithPhone("+79161234567");

    expect(screen.getByTestId("profile-phone-dropped")).toHaveTextContent(
      "profile.phoneDropped:+79161234567",
    );
  });
});
