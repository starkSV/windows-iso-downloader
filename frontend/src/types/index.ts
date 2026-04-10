export interface Sku {
  Id: string
  Language: string
  LocalizedLanguage: string
  FriendlyFileNames?: string[]
}

export interface SkuInfoResponse {
  Skus: Sku[]
}

export interface DownloadOption {
  Uri: string
  Architecture: string
}

export interface ProxyResponse {
  ProductDownloadOptions: DownloadOption[]
}

export interface Product {
  id: string
  name: string
}
