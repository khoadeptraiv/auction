import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { useUser } from '@clerk/clerk-react'
import { getAuction, placeBid, type Auction } from '../services/api'
import useWebSocket from '../hooks/useWebSocket'

export default function AuctionDetail() {
  const { id } = useParams<{ id: string }>()
  const { isSignedIn } = useUser()
  const [auction, setAuction] = useState<Auction | null>(null)
  const [bidAmount, setBidAmount] = useState('')
  const [loading, setLoading] = useState(true)
  const [bidding, setBidding] = useState(false)
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)
  const { isConnected, lastBid } = useWebSocket(id)

  useEffect(() => {
    if (id) loadAuction()
  }, [id])

  useEffect(() => {
    if (lastBid && auction && lastBid.auction_id === auction.id) {
      setAuction(prev => prev ? { ...prev, current_price: lastBid.amount } : null)
      setMessage({ type: 'success', text: `Có người vừa đặt giá ${formatPrice(lastBid.amount)}` })
    }
  }, [lastBid])

  useEffect(() => {
    if (auction) {
      const suggestedBid = auction.current_price + auction.min_increment
      setBidAmount(suggestedBid.toString())
    }
  }, [auction])

  const loadAuction = async () => {
    try {
      const res = await getAuction(id!)
      setAuction(res.data)
    } catch (error) {
      console.error('Failed to load auction:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleBid = async () => {
    if (!auction || !bidAmount) return

    setBidding(true)
    setMessage(null)

    try {
      const res = await placeBid(auction.id, parseFloat(bidAmount))
      if (res.data.success) {
        setMessage({ type: 'success', text: '🎉 Đặt giá thành công!' })
        setAuction(prev => prev ? { ...prev, current_price: res.data.current_price } : null)
        setBidAmount((res.data.current_price + auction.min_increment).toString())
      } else {
        setMessage({ type: 'error', text: res.data.message })
      }
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Đặt giá thất bại' })
    } finally {
      setBidding(false)
    }
  }

  const formatPrice = (price: number) => {
    return new Intl.NumberFormat('vi-VN', { style: 'currency', currency: 'VND' }).format(price)
  }

  const formatDate = (date: string) => {
    return new Date(date).toLocaleString('vi-VN')
  }

  if (loading) return <div>Đang tải...</div>
  if (!auction) return <div>Không tìm thấy phiên đấu giá</div>

  return (
    <div className="auction-detail">
      <div>
        <div className="auction-image">
          {auction.image_url ? (
            <img src={auction.image_url} alt={auction.title} style={{ width: '100%', height: '100%', objectFit: 'cover', borderRadius: '1rem' }} />
          ) : '🏷️'}
        </div>

        <h1 style={{ marginTop: '1.5rem', marginBottom: '1rem' }}>{auction.title}</h1>
        <p style={{ color: 'var(--text-secondary)', marginBottom: '1.5rem' }}>{auction.description || 'Không có mô tả'}</p>

        <div className="card" style={{ padding: '1.5rem' }}>
          <h3 style={{ marginBottom: '1rem' }}>Thông tin phiên đấu giá</h3>
          <div style={{ display: 'grid', gap: '0.5rem' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <span style={{ color: 'var(--text-secondary)' }}>Giá khởi điểm:</span>
              <span>{formatPrice(auction.starting_price)}</span>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <span style={{ color: 'var(--text-secondary)' }}>Bước giá tối thiểu:</span>
              <span>{formatPrice(auction.min_increment)}</span>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <span style={{ color: 'var(--text-secondary)' }}>Thời gian bắt đầu:</span>
              <span>{formatDate(auction.start_time)}</span>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <span style={{ color: 'var(--text-secondary)' }}>Thời gian kết thúc:</span>
              <span>{formatDate(auction.end_time)}</span>
            </div>
          </div>
        </div>
      </div>

      <div>
        <div className="bid-panel">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
            <span className={`status-badge status-${auction.status}`}>{auction.status.toUpperCase()}</span>
            {isConnected && (
              <div className="live-indicator">
                <span className="live-dot"></span>
                LIVE
              </div>
            )}
          </div>

          <div className="current-bid">
            <div className="current-bid-label">Giá hiện tại</div>
            <div className="current-bid-amount">{formatPrice(auction.current_price)}</div>
          </div>

          {auction.status === 'active' && isSignedIn && (
            <>
              <div className="bid-input-group">
                <input
                  type="number"
                  className="form-input"
                  value={bidAmount}
                  onChange={(e) => setBidAmount(e.target.value)}
                  placeholder="Nhập số tiền"
                  min={auction.current_price + auction.min_increment}
                  step={auction.min_increment}
                />
                <button
                  className="btn btn-success btn-lg"
                  onClick={handleBid}
                  disabled={bidding}
                >
                  {bidding ? '...' : '🔨 Đặt giá'}
                </button>
              </div>

              <p style={{ fontSize: '0.875rem', color: 'var(--text-secondary)', textAlign: 'center' }}>
                Tối thiểu: {formatPrice(auction.current_price + auction.min_increment)}
              </p>
            </>
          )}

          {!isSignedIn && auction.status === 'active' && (
            <p style={{ textAlign: 'center', color: 'var(--text-secondary)' }}>
              Đăng nhập để tham gia đấu giá
            </p>
          )}

          {auction.status !== 'active' && (
            <p style={{ textAlign: 'center', color: 'var(--text-secondary)' }}>
              Phiên đấu giá {auction.status === 'pending' ? 'chưa bắt đầu' : 'đã kết thúc'}
            </p>
          )}
        </div>

        {message && (
          <div className={`toast toast-${message.type}`}>{message.text}</div>
        )}
      </div>
    </div>
  )
}
