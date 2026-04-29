import { type RouteConfig, index, route } from "@react-router/dev/routes";

export default [
    route(
        "/.well-known/appspecific/com.chrome.devtools.json",
        "routes/debug-null-route.tsx"
    ),
    route(
        "/obs-overlay",
        "routes/obs-overlay-route.tsx"
    ),
    index("routes/home-route.tsx")
] satisfies RouteConfig;
