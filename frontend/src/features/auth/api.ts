export type AuthTokens = {
  token: string;
  refresh_token: string;
};

export type LoginInput = {
  email: string;
  password: string;
};

export type RegisterInput = LoginInput & {
  firstname: string;
  lastname: string;
};

const apiBaseUrl = import.meta.env.VITE_API_URL ?? '/api';

async function authenticate(path: string, body: LoginInput | RegisterInput): Promise<AuthTokens> {
  const response = await fetch(`${apiBaseUrl}/users/${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });

  if (!response.ok) {
    if (response.status === 403) {
      throw new Error('Your email or password is incorrect.');
    }
    throw new Error('We could not complete your request. Please check your details and try again.');
  }

  return response.json() as Promise<AuthTokens>;
}

export const login = (input: LoginInput) => authenticate('login', input);
export const register = (input: RegisterInput) => authenticate('register', input);
