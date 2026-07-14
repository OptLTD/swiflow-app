/// <reference types="vite/client" />

declare module 'xlsx/dist/cpexcel.full.mjs' {
  const cptable: Record<string, unknown>
  export = cptable
}
