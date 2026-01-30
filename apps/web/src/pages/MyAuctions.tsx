import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { getMyAuctions, type Auction } from '../services/api'

export default function MyAuctions() {
  const [auctions, setAuctions] = useState<Auction[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadAuctions()
  }, [])

  const loadAuctions = async () => {
    try {
      const res = await getMyAuctions()
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

  if (loading) return <div>Đang tải...</div>

  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">📦 Phiên đấu giá của tôi</h1>
        <Link to="/create" className="btn btn-primary">+ Tạo mới</Link>
      </div>

      {auctions.length === 0 ? (
        <div className="card" style={{ padding: '3rem', textAlign: 'center' }}>
          <p style={{ color: 'var(--text-secondary)', marginBottom: '1rem' }}>
            Bạn chưa tạo phiên đấu giá nào
          </p>
          <Link to="/create" className="btn btn-primary">
            Tạo phiên đấu giá đầu tiên
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
                  <span>{new Date(auction.end_time).toLocaleDateString('vi-VN')}</span>
                </div>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}
