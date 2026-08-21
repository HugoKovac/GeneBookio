import { Group, Loader } from '@mantine/core';
import { Navigate, Outlet } from 'react-router-dom';
import { useSubscription } from '../features/subscription/useSubscription';

export default function RequireSubscription() {
  const { isActive, loading } = useSubscription();

  if (loading) {
    return <Group justify="center" mt={48}><Loader /></Group>;
  }

  return isActive ? <Outlet /> : <Navigate to="/subscribe" replace />;
}
