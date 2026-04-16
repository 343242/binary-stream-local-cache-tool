import { render, screen } from "@testing-library/react";

import App from "../App";

test("renders the desktop shell title", () => {
  render(<App />);
  expect(screen.getByText("Binary Stream Cache Tool")).toBeInTheDocument();
});
