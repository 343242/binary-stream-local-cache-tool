import { render, screen } from "@testing-library/react";

import App from "../App";
import styles from "../styles/shell.module.css";

test("renders the desktop shell title", () => {
  render(<App />);
  expect(screen.getByText("二进制流缓存工具")).toBeInTheDocument();
});

test("renders the editorial shell landmarks", () => {
  render(<App />);
  expect(screen.getByRole("complementary", { name: "主工作区" })).toBeInTheDocument();
  expect(screen.getByRole("banner")).toBeInTheDocument();
  expect(screen.getByRole("main")).toBeInTheDocument();
});

test("keeps Home available in the primary navigation", () => {
  render(<App />);
  expect(screen.getByRole("button", { name: "主页" })).toBeInTheDocument();
});

test("keeps the shell readable at narrow desktop widths", () => {
  window.innerWidth = 1180;
  render(<App />);
  expect(screen.getByRole("main")).toHaveClass(styles.frameScrollable);
  expect(screen.getByRole("complementary", { name: "主工作区" })).toBeInTheDocument();
});
