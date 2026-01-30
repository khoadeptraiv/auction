import axios from "axios";

const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080/api";

const api = axios.create({
  baseURL: API_URL,
  headers: {
    "Content-Type": "application/json",
  },
});

// Add Clerk token to requests
export const setAuthToken = (token: string | null) => {
  if (token) {
    api.defaults.headers.common["Authorization"] = `Bearer ${token}`;
  } else {
    delete api.defaults.headers.common["Authorization"];
  }
};

// Types
export interface User {
  id: string;
  email: string;
  full_name: string;
  wallet_balance: number;
}

export interface Auction {
  id: string;
  seller_id: string;
  title: string;
  description: string;
  image_url: string;
  starting_price: number;
  current_price: number;
  min_increment: number;
  start_time: string;
  end_time: string;
  status: "pending" | "active" | "ended";
  winner_id?: string;
  created_at: string;
}

export interface BidResponse {
  success: boolean;
  message: string;
  current_price: number;
  bidder_id?: string;
}

// API functions
export const getAuctions = () =>
  api.get<{ auctions: Auction[]; total: number }>("/auctions");

export const getActiveAuctions = () =>
  api.get<{ auctions: Auction[]; total: number }>("/auctions/active");

export const getAuction = (id: string) => api.get<Auction>(`/auctions/${id}`);

export const createAuction = (data: {
  title: string;
  description: string;
  image_url: string;
  starting_price: number;
  min_increment: number;
  start_time: string;
  end_time: string;
}) => api.post<Auction>("/auctions", data);

export const getMyAuctions = () =>
  api.get<{ auctions: Auction[]; total: number }>("/auctions/my");

export const placeBid = (auctionId: string, amount: number) =>
  api.post<BidResponse>("/bid", { auction_id: auctionId, amount });

export const getWalletBalance = () =>
  api.get<{ balance: number }>("/wallet/balance");

export const deposit = (amount: number) =>
  api.post<{ balance: number }>("/wallet/deposit", { amount });

export const syncUser = (clerkUser: {
  id: string;
  email: string;
  fullName: string;
}) =>
  api.post<User>("/users/sync", {
    clerk_id: clerkUser.id,
    email: clerkUser.email,
    full_name: clerkUser.fullName,
  });

export default api;
