import type { Book } from './types';

const apiBaseUrl = import.meta.env.VITE_API_URL ?? '/api';

async function request(path: string, token: string): Promise<Response> {
  const response = await fetch(`${apiBaseUrl}${path}`, {
    headers: { Authorization: `Bearer ${token}` },
  });

  if (!response.ok) {
    throw new Error('We could not complete your request. Please try again.');
  }

  return response;
}

export async function getBooks(token: string): Promise<Book[]> {
  const response = await request('/books/', token);
  return response.json() as Promise<Book[]>;
}

// getBookAudioStreamURL builds a URL an <audio> element can use directly as
// its src, so the browser streams the file progressively (byte-range
// requests) instead of the app downloading it whole first. A plain <audio>
// src can't carry an Authorization header, so the access token travels as a
// query parameter, which the backend accepts as a fallback for this route.
export function getBookAudioStreamURL(bookID: string, token: string): string {
  return `${apiBaseUrl}/books/audio/${bookID}?token=${encodeURIComponent(token)}`;
}
