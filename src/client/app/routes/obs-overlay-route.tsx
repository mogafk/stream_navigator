import type { Route } from "./+types/home";
import { ObsOverlay } from "../pages/obs-overlay/obs-overlay";

export function meta({}: Route.MetaArgs) {
  return [
    { title: "Stream navigator application" },
    { name: "Overlay for obs", content: "Obs overlay!" },
  ];
}

export default function ObsOverlayRoute() {
  return <ObsOverlay />;
}
