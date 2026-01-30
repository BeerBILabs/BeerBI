import { dirname } from "path";
import { fileURLToPath } from "url";
import { FlatCompat } from "@eslint/eslintrc";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const compat = new FlatCompat({
  baseDirectory: __dirname,
});

const eslintConfig = [
  ...compat.extends("next/core-web-vitals", "next/typescript"),
  {
    files: ["**/next-env.d.ts"],
    rules: {
      "@typescript-eslint/triple-slash-reference": "off",
    },
  },
  {
    rules: {
      "react/no-unescaped-entities": 0,
      "@typescript-eslint/no-explicit-any": 0,
    },
    ignores: [
      "node_modules/**",
      "**/.next/**",
      "out/**",
      "build/**",
      "**/next-env.d.ts",
    ],
  },
];

export default eslintConfig;
