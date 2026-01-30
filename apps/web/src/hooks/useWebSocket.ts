import { useEffect, useRef, useState, useCallback } from "react";

interface BidEvent {
  type: "bid:new";
  auction_id: string;
  amount: number;
  bidder_id: string;
}

interface AuctionEvent {
  type: "auction:start" | "auction:end";
  auction: {
    id: string;
    status: string;
  };
}

type WSEvent = BidEvent | AuctionEvent;

export const useWebSocket = (auctionId?: string) => {
  const ws = useRef<WebSocket | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [lastBid, setLastBid] = useState<BidEvent | null>(null);

  const connect = useCallback(() => {
    const wsUrl = import.meta.env.VITE_WS_URL || "ws://localhost:8084/ws";

    ws.current = new WebSocket(wsUrl);

    ws.current.onopen = () => {
      console.log("WebSocket connected");
      setIsConnected(true);
    };

    ws.current.onmessage = (event) => {
      try {
        const data: WSEvent = JSON.parse(event.data);

        if (data.type === "bid:new") {
          if (!auctionId || data.auction_id === auctionId) {
            setLastBid(data as BidEvent);
          }
        }
      } catch (e) {
        console.error("Failed to parse WebSocket message", e);
      }
    };

    ws.current.onclose = () => {
      console.log("WebSocket disconnected");
      setIsConnected(false);
      // Reconnect after 3 seconds
      setTimeout(connect, 3000);
    };

    ws.current.onerror = (error) => {
      console.error("WebSocket error:", error);
    };
  }, [auctionId]);

  useEffect(() => {
    connect();

    return () => {
      if (ws.current) {
        ws.current.close();
      }
    };
  }, [connect]);

  return { isConnected, lastBid };
};

export default useWebSocket;
