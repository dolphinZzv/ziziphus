import { Component, type ReactNode } from 'react'
import { withTranslation, type TFunction } from 'react-i18next'
import { AlertCircle, RefreshCw } from 'lucide-react'

interface Props { children: ReactNode; fallback?: ReactNode; t: TFunction }

interface State { hasError: boolean; error: Error | null }

class ErrorBoundaryBase extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null })
  }

  render() {
    const { t } = this.props
    if (this.state.hasError) {
      if (this.props.fallback) return this.props.fallback
      return (
        <div className="h-full flex items-center justify-center bg-[var(--color-canvas)]">
          <div className="text-center space-y-3 max-w-sm">
            <AlertCircle size={32} className="mx-auto text-[var(--destructive)] opacity-60" />
            <h3 className="font-headline text-sm font-semibold text-[var(--color-ink)]">{t('error.pageError')}</h3>
            <p className="text-xs text-[var(--color-muted)]">{this.state.error?.message || t('error.unknown')}</p>
            <button
              onClick={this.handleRetry}
              className="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white text-sm transition-colors"
            >
              <RefreshCw size={14} /> {t('common.retry')}
            </button>
          </div>
        </div>
      )
    }
    return this.props.children
  }
}

export default withTranslation()(ErrorBoundaryBase)
