import { Outlet, Link, useLocation } from 'react-router-dom'
import { SignedIn, SignedOut, UserButton } from '@clerk/clerk-react'
import { useEffect } from 'react'
import { useAuth } from '@clerk/clerk-react'
import { setAuthToken } from '../services/api'

export default function Layout() {
  const location = useLocation()
  const { getToken } = useAuth()

  useEffect(() => {
    const updateToken = async () => {
      const token = await getToken()
      setAuthToken(token)
    }
    updateToken()
  }, [getToken])

  const isActive = (path: string) => location.pathname === path ? 'nav-link active' : 'nav-link'

  return (
    <div className="layout">
      <nav className="navbar">
        <div className="navbar-content">
          <Link to="/" className="logo">
            🔨 AuctionHub
          </Link>

          <div className="nav-links">
            <Link to="/" className={isActive('/')}>Đấu giá</Link>
            
            <SignedIn>
              <Link to="/create" className={isActive('/create')}>Tạo mới</Link>
              <Link to="/my-auctions" className={isActive('/my-auctions')}>Của tôi</Link>
              <Link to="/wallet" className={isActive('/wallet')}>Ví tiền</Link>
              <UserButton afterSignOutUrl="/" />
            </SignedIn>

            <SignedOut>
              <Link to="/sign-in" className="btn btn-primary">Đăng nhập</Link>
            </SignedOut>
          </div>
        </div>
      </nav>

      <main className="main-content">
        <Outlet />
      </main>
    </div>
  )
}
