export function StyleStep() {
  return (
    <div className="flex flex-col gap-5">
      <p className="text-sm text-text-muted">
        Customise the look of your CTF platform. Logo, banner, and favicon upload will be available
        in the next release.
      </p>

      {(['Logo', 'Banner', 'Favicon'] as const).map((label) => (
        <div key={label} className="flex flex-col gap-1.5">
          <span className="text-sm font-medium text-text-secondary">{label}</span>
          <div className="flex items-center justify-between p-4 rounded-lg border-2 border-dashed border-border-muted bg-deep-space/30 opacity-50 cursor-not-allowed select-none">
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded-md bg-surface-muted flex items-center justify-center">
                <svg
                  className="w-4 h-4 text-text-muted"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={1.5}
                    d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
                  />
                </svg>
              </div>
              <span className="text-sm text-text-muted">Click to upload {label.toLowerCase()}</span>
            </div>
            <span className="text-xs font-medium px-2 py-0.5 rounded-full bg-surface-muted text-text-muted border border-border-muted">
              Coming soon
            </span>
          </div>
        </div>
      ))}
    </div>
  )
}
