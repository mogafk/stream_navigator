import type { Route } from "./+types/home";

export function meta({}: Route.MetaArgs) {
  return [
    { title: "Null page" },
    { name: "description", content: "Null page" },
  ];
}

export default function DebugNullRoute() {
  return null;
}
