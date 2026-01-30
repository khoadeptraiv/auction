import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { createAuction } from '../services/api'

export default function CreateAuction() {
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const [form, setForm] = useState({
    title: '',
    description: '',
    image_url: '',
    starting_price: '',
    min_increment: '10000',
    start_time: '',
    end_time: '',
  })

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')

    try {
      const res = await createAuction({
        title: form.title,
        description: form.description,
        image_url: form.image_url,
        starting_price: parseFloat(form.starting_price),
        min_increment: parseFloat(form.min_increment),
        start_time: new Date(form.start_time).toISOString(),
        end_time: new Date(form.end_time).toISOString(),
      })
      navigate(`/auction/${res.data.id}`)
    } catch (err: any) {
      setError(err.response?.data?.error || 'Tạo phiên đấu giá thất bại')
    } finally {
      setLoading(false)
    }
  }

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    setForm(prev => ({ ...prev, [e.target.name]: e.target.value }))
  }

  return (
    <div style={{ maxWidth: '600px', margin: '0 auto' }}>
      <h1 className="page-title" style={{ marginBottom: '2rem' }}>Tạo phiên đấu giá mới</h1>

      <form onSubmit={handleSubmit} className="card" style={{ padding: '2rem' }}>
        <div className="form-group">
          <label className="form-label">Tiêu đề *</label>
          <input
            type="text"
            name="title"
            className="form-input"
            value={form.title}
            onChange={handleChange}
            placeholder="VD: iPhone 15 Pro Max 256GB"
            required
          />
        </div>

        <div className="form-group">
          <label className="form-label">Mô tả</label>
          <textarea
            name="description"
            className="form-input"
            value={form.description}
            onChange={handleChange}
            placeholder="Mô tả chi tiết sản phẩm..."
            rows={4}
            style={{ resize: 'vertical' }}
          />
        </div>

        <div className="form-group">
          <label className="form-label">URL hình ảnh</label>
          <input
            type="url"
            name="image_url"
            className="form-input"
            value={form.image_url}
            onChange={handleChange}
            placeholder="https://example.com/image.jpg"
          />
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem' }}>
          <div className="form-group">
            <label className="form-label">Giá khởi điểm (VND) *</label>
            <input
              type="number"
              name="starting_price"
              className="form-input"
              value={form.starting_price}
              onChange={handleChange}
              placeholder="1000000"
              min="1000"
              required
            />
          </div>

          <div className="form-group">
            <label className="form-label">Bước giá tối thiểu (VND)</label>
            <input
              type="number"
              name="min_increment"
              className="form-input"
              value={form.min_increment}
              onChange={handleChange}
              placeholder="10000"
              min="1000"
            />
          </div>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem' }}>
          <div className="form-group">
            <label className="form-label">Thời gian bắt đầu *</label>
            <input
              type="datetime-local"
              name="start_time"
              className="form-input"
              value={form.start_time}
              onChange={handleChange}
              required
            />
          </div>

          <div className="form-group">
            <label className="form-label">Thời gian kết thúc *</label>
            <input
              type="datetime-local"
              name="end_time"
              className="form-input"
              value={form.end_time}
              onChange={handleChange}
              required
            />
          </div>
        </div>

        {error && (
          <div style={{ color: 'var(--danger)', marginBottom: '1rem', textAlign: 'center' }}>
            {error}
          </div>
        )}

        <button type="submit" className="btn btn-primary btn-lg" style={{ width: '100%' }} disabled={loading}>
          {loading ? 'Đang tạo...' : '🚀 Tạo phiên đấu giá'}
        </button>
      </form>
    </div>
  )
}
