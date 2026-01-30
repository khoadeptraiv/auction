import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { getActiveAuctions, type Auction } from '../services/api'
import useWebSocket from '../hooks/useWebSocket'

export default function Home() {
  const [auctions, setAuctions] = useState<Auction[]>([])
  const [loading, setLoading] = useState(true)
  const { isConnected, lastBid } = useWebSocket()

  useEffect(() => {
    loadAuctions()
  }, [])

  useEffect(() => {
    if (lastBid) {
      setAuctions(prev => prev.map(a => 
        a.id === lastBid.auction_id 
          ? { ...a, current_price: lastBid.amount }
          : a
      ))
    }
  }, [lastBid])

  const loadAuctions = async () => {
    try {
      const res = await getActiveAuctions()
      setAuctions(res.data.auctions || [])
    } catch (error) {
      console.error('Failed to load auctions:', error)
    } finally {
      setLoading(false)
    }
  }

  const formatPrice = (price: number) => {
    return new Intl.NumberFormat('vi-VN', { style: 'currency', currency: 'VND' }).format(price)
  }

  const formatTimeLeft = (endTime: string) => {
    const end = new Date(endTime)
    const now = new Date()
    const diff = end.getTime() - now.getTime()
    
    if (diff <= 0) return 'Đã kết thúc'
    
    const hours = Math.floor(diff / (1000 * 60 * 60))
    const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60))
    
    if (hours > 24) {
      const days = Math.floor(hours / 24)
      return `Còn ${days} ngày`
    }
    
    return `Còn ${hours}h ${minutes}m`
  }

  if (loading) {
    return <div className="loading">Đang tải...</div>
  }

  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">🔥 Đấu giá đang diễn ra</h1>
        {isConnected && (
          <div className="live-indicator">
            <span className="live-dot"></span>
            LIVE
          </div>
        )}
      </div>

      {auctions.length === 0 ? (
        <div className="card" style={{ padding: '3rem', textAlign: 'center' }}>
          <p style={{ color: 'var(--text-secondary)', marginBottom: '1rem' }}>
            Chưa có phiên đấu giá nào đang diễn ra
          </p>
          <Link to="/create" className="btn btn-primary">
            Tạo phiên đấu giá mới
          </Link>
        </div>
      ) : (
        <div className="auction-grid">
          {auctions.map(auction => (
            <Link to={`/auction/${auction.id}`} key={auction.id} className="card" style={{ textDecoration: 'none' }}>
              <div className="auction-card-image">
                {auction.image_url ? (
                  <img src={auction.image_url} alt={auction.title} style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
                ) : '🏷️'}
              </div>
              <div className="auction-card-content">
                <h3 className="auction-card-title">{auction.title}</h3>
                <div className="auction-card-price">{formatPrice(auction.current_price)}</div>
                <div className="auction-card-meta">
                  <span className={`status-badge status-${auction.status}`}>{auction.status}</span>
                  <span>{formatTimeLeft(auction.end_time)}</span>
                </div>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}
