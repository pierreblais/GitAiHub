"use client";

import { useEffect, useState } from "react";

/** "Good morning, pierre!" in the viewer's own time zone. */
export function Greeting({ name }: { name: string }) {
  const [part, setPart] = useState("Welcome back");
  useEffect(() => {
    const h = new Date().getHours();
    setPart(h < 5 ? "Good evening" : h < 12 ? "Good morning" : h < 18 ? "Good afternoon" : "Good evening");
  }, []);
  return (
    <>
      {part}, {name}!
    </>
  );
}
