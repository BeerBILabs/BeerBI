import nextPlugin from "@next/eslint-plugin-next";
import reactHooksPlugin from "eslint-plugin-react-hooks";
import tseslint from "typescript-eslint";

const eslintConfig = [
  {
    ignores: [
      "node_modules/**",
      ".next/**",
      "out/**",
      "build/**",
      "coverage/**",
    ],
  },
  // typescript-eslint: recommended
  ...tseslint.configs.recommended,
  // Next.js: recommended + core web vitals (flat configs, register the "@next/next" plugin)
  nextPlugin.configs["core-web-vitals"],
  // React hooks: rules-of-hooks, exhaustive-deps and the React Compiler based rules
  reactHooksPlugin.configs.flat.recommended,
  {
    files: ["**/next-env.d.ts"],
    rules: {
      "@typescript-eslint/triple-slash-reference": "off",
    },
  },
];

export default eslintConfig;
