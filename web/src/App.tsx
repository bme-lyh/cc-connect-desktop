import { useEffect, useState } from 'react';
import { Routes, Route, Navigate, useLocation, useNavigate } from 'react-router-dom';
import { useAuthStore } from '@/store/auth';
import Layout from '@/components/Layout/Layout';
import Login from '@/pages/Login';
import BotList from '@/pages/Bots/BotList';
import SetupWizard from '@/pages/Setup/SetupWizard';
import { getSetupStatus } from '@/api/bots';
import ProjectList from '@/pages/Projects/ProjectList';
import ProjectDetail from '@/pages/Projects/ProjectDetail';
import ChatList from '@/pages/Chat/ChatList';
import ChatView from '@/pages/Chat/ChatView';
import CronList from '@/pages/Cron/CronList';
import SystemConfig from '@/pages/System/Config';
import ProviderList from '@/pages/Providers/ProviderList';
import SkillList from '@/pages/Skills/SkillList';

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

function SetupGate({ children }: { children: React.ReactNode }) {
  const location = useLocation();
  const navigate = useNavigate();
  const [checking, setChecking] = useState(true);

  useEffect(() => {
    getSetupStatus()
      .then(status => {
        if (status.first_run && location.pathname !== '/setup') navigate('/setup', { replace: true });
      })
      .catch(() => undefined)
      .finally(() => setChecking(false));
  }, [location.pathname, navigate]);

  if (checking) return <div className="min-h-screen flex items-center justify-center text-gray-400">Loading...</div>;
  return <>{children}</>;
}

export default function App() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);

  return (
    <Routes>
      <Route path="/login" element={isAuthenticated ? <Navigate to="/" replace /> : <Login />} />
      <Route path="/setup" element={<ProtectedRoute><SetupWizard /></ProtectedRoute>} />
      <Route element={<ProtectedRoute><SetupGate><Layout /></SetupGate></ProtectedRoute>}>
        <Route index element={<BotList />} />
        <Route path="projects" element={<ProjectList />} />
        <Route path="projects/:name" element={<ProjectDetail />} />
        <Route path="providers" element={<ProviderList />} />
        <Route path="skills" element={<SkillList />} />
        <Route path="chat" element={<ChatList />} />
        <Route path="chat/:name" element={<ChatView />} />
        <Route path="cron" element={<CronList />} />
        <Route path="system" element={<SystemConfig />} />
      </Route>
    </Routes>
  );
}
