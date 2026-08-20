import type { CatalogBook, Language, SearchResult } from './types';

const apiBaseUrl = import.meta.env.VITE_API_URL ?? '/api';

async function request(path: string, init?: RequestInit): Promise<Response> {
  const response = await fetch(`${apiBaseUrl}${path}`, init);

  if (!response.ok) {
    const body = await response.text();
    throw new Error(body || `Request failed (${response.status})`);
  }

  return response;
}

export async function searchBooks(query: string, signal?: AbortSignal): Promise<SearchResult[]> {
  const response = await request(`/search?q=${encodeURIComponent(query)}`, { signal });
  return (await response.json()) as SearchResult[];
}

export async function getCatalog(): Promise<CatalogBook[]> {
  const response = await request('/books/');
  return (await response.json()) as CatalogBook[];
}

export async function uploadBook(bookKey: string, language: Language, epub: File): Promise<void> {
  const form = new FormData();
  form.append('book_key', bookKey);
  form.append('language', language);
  form.append('epub', epub);

  await request('/upload', { method: 'POST', body: form });
}

export async function retryBook(bookID: string): Promise<void> {
  await request(`/books/${bookID}/retry`, { method: 'POST' });
}
