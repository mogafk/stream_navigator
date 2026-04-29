import type { Route } from "./+types/home";
import { Settings } from "../pages/settings/settings";

export function meta({}: Route.MetaArgs) {
  return [
    { title: "Stream navigator application" },
    { name: "Settings", content: "Settings for stream navigator" },
  ];
}

export default function HomeRoute() {
  return <Settings />;
}
