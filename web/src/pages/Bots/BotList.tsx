import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import { Bot, FolderOpen, Plus, Power, RefreshCw, Settings2, Trash2 } from 'lucide-react';
import { Badge, Button, Card, EmptyState } from '@/components/ui';
import { listBots, removeBot, updateBotState, waitUntilReady, type BotSummary } from '@/api/bots';

export default function BotList() {
  const { t } = useTranslation();
  const [bots, setBots] = useState<BotSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState('');
  const [error, setError] = useState('');

  const refresh = useCallback(async () => {
    try {
      setError('');
      const response = await listBots();
      setBots(response.bots || []);
    } catch (e: any) {
      setError(e?.message || String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { refresh(); }, [refresh]);

  const apply = async (bot: BotSummary, enabled: boolean) => {
    setBusy(bot.id);
    setError('');
    try {
      await updateBotState(bot.id, enabled);
      await waitUntilReady();
      await refresh();
    } catch (e: any) {
      setError(e?.message || String(e));
    } finally {
      setBusy('');
    }
  };

  const remove = async (bot: BotSummary) => {
    if (!window.confirm(t('simple.confirmRemove', { name: bot.display_name }))) return;
    setBusy(bot.id);
    try {
      await removeBot(bot.id);
      await waitUntilReady();
      await refresh();
    } catch (e: any) {
      setError(e?.message || String(e));
    } finally {
      setBusy('');
    }
  };

  const reconnect = async (bot: BotSummary) => {
    setBusy(bot.id);
    setError('');
    try {
      await updateBotState(bot.id, false);
      await waitUntilReady();
      await updateBotState(bot.id, true);
      await waitUntilReady();
      await refresh();
    } catch (e: any) {
      setError(e?.message || String(e));
    } finally {
      setBusy('');
    }
  };

  if (loading) return <div className="h-64 flex items-center justify-center text-gray-400"><RefreshCw className="animate-spin" /></div>;

  return (
    <div className="space-y-5 animate-fade-in">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h2 className="text-xl font-bold text-gray-900 dark:text-white">{t('simple.myBots')}</h2>
          <p className="text-sm text-gray-500 mt-1">{t('simple.botSubtitle')}</p>
        </div>
        <Link to="/setup"><Button size="sm"><Plus size={15} /> {t('simple.addBot')}</Button></Link>
      </div>

      {error && <div className="rounded-xl bg-red-50 dark:bg-red-950/30 text-red-600 p-3 text-sm">{error}</div>}

      {bots.length === 0 ? (
        <Card className="py-12"><EmptyState message={t('simple.noBots')} icon={Bot} /></Card>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-4">
          {bots.map(bot => (
            <Card key={bot.id} className="flex flex-col gap-4">
              <div className="flex items-start justify-between gap-3">
                <div className="flex items-center gap-3 min-w-0">
                  <div className="w-11 h-11 rounded-xl bg-accent/10 flex items-center justify-center shrink-0"><Bot size={21} className="text-accent" /></div>
                  <div className="min-w-0">
                    <h3 className="font-semibold text-gray-900 dark:text-white truncate">{bot.display_name}</h3>
                    <p className="text-xs text-gray-400 truncate">{bot.platform_type} · {bot.agent_type}</p>
                  </div>
                </div>
                <Badge variant={bot.runtime_state === 'running' ? 'success' : bot.runtime_state === 'error' ? 'danger' : 'default'}>{t(`simple.state.${bot.runtime_state}`)}</Badge>
              </div>
              <div className="space-y-2 text-xs text-gray-500 dark:text-gray-400">
                <div className="flex items-center gap-2"><FolderOpen size={13} /><span className="truncate" title={bot.work_dir}>{bot.work_dir}</span></div>
                {bot.runtime_error && <p className="rounded-lg bg-red-50 dark:bg-red-950/30 px-2.5 py-2 text-red-600">{bot.runtime_error}</p>}
                <div className="flex flex-wrap gap-1.5">
                  {bot.model && <Badge>{bot.model}</Badge>}
                  {bot.reasoning_effort && <Badge>{bot.reasoning_effort}</Badge>}
                  <Badge>{bot.permission_mode || 'default'}</Badge>
                </div>
              </div>
              <div className="mt-auto pt-3 border-t border-gray-100 dark:border-white/[0.06] flex items-center gap-2">
                {bot.runtime_state === 'error' && bot.enabled
                  ? <Button size="sm" loading={busy === bot.id} onClick={() => reconnect(bot)}><RefreshCw size={13} /> {t('simple.reconnect')}</Button>
                  : <Button size="sm" variant={bot.enabled ? 'secondary' : 'primary'} loading={busy === bot.id} onClick={() => apply(bot, !bot.enabled)}><Power size={13} /> {bot.enabled ? t('simple.stop') : t('simple.start')}</Button>}
                <Link to={`/setup?edit=${encodeURIComponent(bot.id)}`}><Button size="sm" variant="secondary"><Settings2 size={13} /> {t('common.settings', 'Settings')}</Button></Link>
                <button className="ml-auto p-2 text-gray-400 hover:text-red-500" onClick={() => remove(bot)} aria-label={t('common.delete')}><Trash2 size={15} /></button>
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
