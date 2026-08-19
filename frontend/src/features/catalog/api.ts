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

// getBookAudioURL fetches the protected audio stream and exposes it as an
// object URL, since a plain <audio> src can't carry an Authorization header.
export async function getBookAudioURL(bookID: string, token: string): Promise<string> {
  const response = await request(`/books/audio/${bookID}`, token);
  const blob = await response.blob();
  return URL.createObjectURL(blob);
}
