import type { Metadata } from "next";
import { LogoMark } from "@/components/Logo";
import { LoginForm } from "./LoginForm";

export const metadata: Metadata = { title: "Sign in" };

export default async function LoginPage({ searchParams }: { searchParams: Promise<{ next?: string }> }) {
  const { next } = await searchParams;
  return (
    <main className="gh-login px-3">
      <div className="text-center mb-4">
        <LogoMark size={48} />
        <h1 className="f2 text-normal mt-3">Sign in to GitaiHub</h1>
        <p className="color-fg-muted f6">The web front end for your gitai server.</p>
      </div>
      <LoginForm next={next?.startsWith("/") ? next : "/"} />
    </main>
  );
}
