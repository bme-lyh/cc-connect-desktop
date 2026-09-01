import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import Login from './Login';
import { useAuthStore } from '@/store/auth';

vi.mock('@/api/status', () => ({ getStatus: vi.fn().mockResolvedValue({}) }));

describe('Login', () => {
  beforeEach(() => {
    localStorage.clear();
    useAuthStore.getState().logout();
  });

  it('uses the token query parameter for one-click desktop login', async () => {
    render(
      <MemoryRouter initialEntries={['/login?token=desktop-token']}>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/" element={<div>setup-home</div>} />
        </Routes>
      </MemoryRouter>,
    );

    await screen.findByText('setup-home');
    await waitFor(() => expect(localStorage.getItem('cc_token')).toBe('desktop-token'));
  });
});
