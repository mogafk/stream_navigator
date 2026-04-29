import { type RouteConfig, index, route } from "@react-router/dev/routes";

export default [
    route(
        "/.well-known/appspecific/com.chrome.devtools.json",
        "routes/debug-null.tsx"
    ),
    index("routes/home.tsx")
] satisfies RouteConfig;
