type BrandMarkProps = {
  size?: 'full' | 'compact'
}

const brandAssets = {
  full: '/brand/myflowhub-symbol-v5.svg',
  compact: '/brand/myflowhub-symbol-v5-compact.svg',
} as const

export function BrandMark({ size = 'full' }: BrandMarkProps) {
  return <img className={`brand-mark ${size === 'compact' ? 'small' : ''}`} src={brandAssets[size]} alt="" aria-hidden="true" draggable={false} />
}
