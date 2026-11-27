import type { Metadata, Viewport } from "next";
import "@primer/css/dist/primer.css";
import "@primer/primitives/dist/css/functional/themes/light.css";
import "@primer/primitives/dist/css/functional/themes/dark.css";
import "./globals.css";

export const metadata: Metadata = {
  title: { default: "GitaiHub", template: "%s · GitaiHub" },
  description: "A web front end for gitai, the Git server for AI coding agents.",
  icons: { icon: "/icon.svg" },
};

export const viewport: Viewport = { width: "device-width", initialScale: 1, themeColor: "#1c1433" };

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" data-color-mode="auto" data-light-theme="light" data-dark-theme="dark">
      <body>{children}</body>
    </html>
  );
}
