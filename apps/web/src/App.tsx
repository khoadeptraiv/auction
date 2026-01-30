import { Routes, Route, Navigate } from 'react-router-dom'
import { SignedIn, SignedOut, SignIn, SignUp } from '@clerk/clerk-react'
import Layout from './components/Layout'
import Home from './pages/Home'
import AuctionDetail from './pages/AuctionDetail'
import CreateAuction from './pages/CreateAuction'
import Wallet from './pages/Wallet'
import MyAuctions from './pages/MyAuctions'

function App() {
  return (
    <Routes>
      {/* Public routes */}
      <Route path="/sign-in/*" element={
        <div className="auth-page">
          <SignIn routing="path" path="/sign-in" signUpUrl="/sign-up" />
        </div>
      } />
      <Route path="/sign-up/*" element={
        <div className="auth-page">
          <SignUp routing="path" path="/sign-up" signInUrl="/sign-in" />
        </div>
      } />

      {/* Protected routes */}
      <Route element={<Layout />}>
        <Route path="/" element={<Home />} />
        <Route path="/auction/:id" element={<AuctionDetail />} />
        
        <Route path="/create" element={
          <>
            <SignedIn><CreateAuction /></SignedIn>
            <SignedOut><Navigate to="/sign-in" /></SignedOut>
          </>
        } />
        
        <Route path="/wallet" element={
          <>
            <SignedIn><Wallet /></SignedIn>
            <SignedOut><Navigate to="/sign-in" /></SignedOut>
          </>
        } />
        
        <Route path="/my-auctions" element={
          <>
            <SignedIn><MyAuctions /></SignedIn>
            <SignedOut><Navigate to="/sign-in" /></SignedOut>
          </>
        } />
      </Route>
    </Routes>
  )
}

export default App
