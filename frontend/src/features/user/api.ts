import { authFetch } from '../../lib/apiClient';
import type { User } from './types';

export type UpdateUserInput = {
  firstname: string;
  lastname: string;
  language: 'en' | 'fr';
};

async function request(path: string, token: string, init?: RequestInit): Promise<Response> {
  const response = await authFetch(path, token, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...init?.headers },
  });

  if (!response.ok) {
    throw new Error('We could not complete your request. Please try again.');
  }

  return response;
}

export async function getUser(id: string, token: string): Promise<User> {
  const response = await request(`/users/${id}`, token);
  return response.json() as Promise<User>;
}

export async function updateUser(id: string, token: string, input: UpdateUserInput): Promise<User> {
  const response = await request(`/users/${id}`, token, { method: 'PATCH', body: JSON.stringify(input) });
  return response.json() as Promise<User>;
}

export async function deleteUser(id: string, token: string): Promise<void> {
  await request(`/users/${id}`, token, { method: 'DELETE' });
}
