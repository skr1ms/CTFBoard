/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_URL?: string
  readonly VITE_HOST?: string
  readonly VITE_FILER_PORT?: string
  readonly VITE_MASTER_PORT?: string
  readonly VITE_MASTER_PROXY_PATH?: string
  readonly VITE_FILER_PROXY_PATH?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
