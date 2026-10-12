const uploadedImagePath = /^\/api\/images\/[0-9a-f-]{36}$/i

// Uploads are stored lossless at up to 2560px; cards and previews only need the 800px rendition.
export function thumbnailSrc<T extends string | null | undefined>(src: T): T | string {
  return src && uploadedImagePath.test(src) ? `${src}/thumb` : src
}
