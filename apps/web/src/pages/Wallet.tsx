import { useState, useEffect } from 'react'
import { getWalletBalance, deposit } from '../services/api'

export default function Wallet() {
  const [balance, setBalance] = useState(0)
  const [loading, setLoading] = useState(true)
  const [depositAmount, setDepositAmount] = useState('')
  const [depositing, setDepositing] = useState(false)
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)

  useEffect(() => {
    loadBalance()
  }, [])

  const loadBalance = async () => {
    try {
      const res = await getWalletBalance()
      setBalance(res.data.balance)
    } catch (error) {
      console.error('Failed to load balance:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleDeposit = async () => {
    if (!depositAmount || parseFloat(depositAmount) <= 0) return

    setDepositing(true)
    setMessage(null)

    try {
      const res = await deposit(parseFloat(depositAmount))
      setBalance(res.data.balance)
      setDepositAmount('')
      setMessage({ type: 'success', text: '✅ Nạp tiền thành công!' })
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Nạp tiền thất bại' })
    } finally {
      setDepositing(false)
    }
  }

  const formatPrice = (price: number) => {
    return new Intl.NumberFormat('vi-VN', { style: 'currency', currency: 'VND' }).format(price)
  }

  const quickAmounts = [100000, 500000, 1000000, 5000000]

  if (loading) return <div>Đang tải...</div>

  return (
    <div style={{ maxWidth: '500px', margin: '0 auto' }}>
      <h1 className="page-title" style={{ marginBottom: '2rem' }}>💰 Ví của tôi</h1>

      <div className="wallet-balance">
        <div className="wallet-balance-label">Số dư hiện tại</div>
        <div className="wallet-balance-amount">{formatPrice(balance)}</div>
      </div>

      <div className="card" style={{ padding: '1.5rem' }}>
        <h3 style={{ marginBottom: '1rem' }}>Nạp tiền</h3>

        <div className="form-group">
          <label className="form-label">Số tiền (VND)</label>
          <input
            type="number"
            className="form-input"
            value={depositAmount}
            onChange={(e) => setDepositAmount(e.target.value)}
            placeholder="Nhập số tiền muốn nạp"
            min="10000"
          />
        </div>

        <div style={{ display: 'flex', gap: '0.5rem', marginBottom: '1.5rem', flexWrap: 'wrap' }}>
          {quickAmounts.map(amount => (
            <button
              key={amount}
              type="button"
              className="btn btn-secondary"
              onClick={() => setDepositAmount(amount.toString())}
            >
              {formatPrice(amount)}
            </button>
          ))}
        </div>

        <button
          className="btn btn-success btn-lg"
          style={{ width: '100%' }}
          onClick={handleDeposit}
          disabled={depositing || !depositAmount}
        >
          {depositing ? 'Đang xử lý...' : '💳 Nạp tiền'}
        </button>

        <p style={{ marginTop: '1rem', fontSize: '0.875rem', color: 'var(--text-secondary)', textAlign: 'center' }}>
          * Đây là demo, tiền sẽ được nạp trực tiếp vào ví
        </p>
      </div>

      {message && (
        <div className={`toast toast-${message.type}`}>{message.text}</div>
      )}
    </div>
  )
}
