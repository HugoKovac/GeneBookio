import { createContext } from 'react';
import type { LoginInput, RegisterInput } from './api';

export type AuthContextValue = {
  isAuthenticated: boolean;
  userID: string | null;
  token: string | null;
  login: (input: LoginInput) => Promise<void>;
  register: (input: RegisterInput) => Promise<void>;
  logout: () => void;
};

export const AuthContext = createContext<AuthContextValue | null>(null);
