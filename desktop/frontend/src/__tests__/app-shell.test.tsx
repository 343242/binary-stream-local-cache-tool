import { render, screen } from "@testing-library/react";

import App from "../App";

test("renders the desktop shell title", () => {
  render(<App />);
  expect(screen.getByText("Binary Stream Cache Tool")).toBeInTheDocument();
});

test("renders the editorial shell landmarks", () => {
  render(<App />);
  expect(screen.getByRole("complementary", { name: /primary workspace/i })).toBeInTheDocument();
  expect(screen.getByRole("banner")).toBeInTheDocument();
  expect(screen.getByRole("main")).toBeInTheDocument();
});
