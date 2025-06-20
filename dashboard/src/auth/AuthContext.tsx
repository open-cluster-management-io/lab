import { createContext, useState, useContext, type ReactNode, useEffect } from 'react';



interface AuthContextType {
  token: string | null;
  isAuthenticated: boolean;
  login: (token: string) => void;
  logout: () => void;
  isLoading: boolean;
  error: string | null;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [token, setToken] = useState<string | null>(() => {
    return localStorage.getItem('authToken');
  });

  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const isAuthenticated = !!token;

  useEffect(() => {
    if (token) {
      localStorage.setItem('authToken', token);
    } else {
      localStorage.removeItem('authToken');
    }
  }, [token]);



  const login = (newToken: string) => {
    setError(null);
    // Clean and format the token
    const cleanToken = newToken.trim();
    const formattedToken = cleanToken.startsWith('Bearer ')
      ? cleanToken
      : `Bearer ${cleanToken}`;

    setToken(formattedToken);
    setIsLoading(false);
  };

  const logout = () => {
    setToken(null);
    setError(null);
    setIsLoading(false);
  };

  const value = {
    token,
    isAuthenticated,
    login,
    logout,
    isLoading,
    error
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};