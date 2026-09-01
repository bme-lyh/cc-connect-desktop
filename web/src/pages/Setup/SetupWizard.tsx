import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { QRCodeSVG } from 'qrcode.react';
import { Bot, Check, CheckCircle2, ChevronDown, ChevronLeft, ChevronRight, FolderOpen, Loader2, RefreshCw, ShieldCheck, XCircle } from 'lucide-react';
import { Button, Card, Input } from '@/components/ui';
import { createBot, getSetupCatalog, getSetupModels, getSetupStatus, listBots, selectSetupDirectory, updateBot, waitUntilReady, type AgentHealth, type SetupCatalog, type SetupField, type SetupModel, type SetupPlatform } from '@/api/bots';
import { setupFeishuBegin, setupFeishuPoll, setupWeixinBegin, setupWeixinPoll } from '@/api/setup';

type Values = Record<string, any>;

export default function SetupWizard() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const editID = searchParams.get('edit') || '';
  const [step, setStep] = useState(1);
  const [catalog, setCatalog] = useState<SetupCatalog | null>(null);
  const [health, setHealth] = useState<AgentHealth[]>([]);
  const [advancedAgents, setAdvancedAgents] = useState(false);
  const [displayName, setDisplayName] = useState('');
  const [agentType, setAgentType] = useState('codex');
  const [workDir, setWorkDir] = useState('');
  const [permissionMode, setPermissionMode] = useState('default');
  const [model, setModel] = useState('');
  const [models, setModels] = useState<SetupModel[]>([]);
  const [modelsLoading, setModelsLoading] = useState(false);
  const [modelError, setModelError] = useState('');
  const [directoryPicking, setDirectoryPicking] = useState(false);
  const [reasoningEffort, setReasoningEffort] = useState('medium');
  const [replyFooter, setReplyFooter] = useState(false);
  const [thinkingMessages, setThinkingMessages] = useState(true);
  const [toolMessages, setToolMessages] = useState(true);
  const [platformType, setPlatformType] = useState('');
  const [originalPlatformType, setOriginalPlatformType] = useState('');
  const [options, setOptions] = useState<Values>({});
  const [showAdvancedFields, setShowAdvancedFields] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    Promise.all([getSetupCatalog(), getSetupStatus(), editID ? listBots() : Promise.resolve({ bots: [] })])
      .then(([nextCatalog, status, botResponse]) => {
        setCatalog(nextCatalog);
        setHealth(status.agents || []);
        const editing = botResponse.bots.find(bot => bot.id === editID);
        if (editing) {
          setDisplayName(editing.display_name);
          setAgentType(editing.agent_type);
          setWorkDir(editing.work_dir);
          setPermissionMode(editing.permission_mode || 'default');
          setModel(editing.model || '');
          setReasoningEffort(editing.reasoning_effort || 'medium');
          setReplyFooter(editing.reply_footer);
          setThinkingMessages(editing.thinking_messages ?? true);
          setToolMessages(editing.tool_messages ?? true);
          setPlatformType(editing.platform_type);
          setOriginalPlatformType(editing.platform_type);
        } else {
          const preferred = nextCatalog.agents.find(a => a.key === 'codex') || nextCatalog.agents[0];
          if (preferred) setAgentType(preferred.key);
        }
      })
      .catch(e => setError(e?.message || String(e)));
  }, [editID]);

  const agents = useMemo(() => catalog?.agents.filter(a => advancedAgents || a.recommended) || [], [catalog, advancedAgents]);
  const selectedPlatform = catalog?.platforms.find(p => p.key === platformType);
  const selectedAgent = catalog?.agents.find(agent => agent.key === agentType);
  const selectedHealth = health.find(a => a.key === agentType);
  const agentBlocked = !!selectedHealth && (!selectedHealth.installed || !selectedHealth.logged_in);

  useEffect(() => {
    const modes = selectedAgent?.modes || [];
    if (modes.length > 0 && !modes.includes(permissionMode)) setPermissionMode(modes[0]);
  }, [selectedAgent, permissionMode]);

  const loadModels = useCallback(async () => {
    if (!agentType) return;
    setModelsLoading(true);
    setModelError('');
    try {
      const response = await getSetupModels(agentType);
      const nextModels = response.models || [];
      setModels(nextModels);
      setModel(current => {
        // Preserve an existing configured model even when local discovery no
        // longer reports it. modelOptions keeps that value selectable so
        // editing an older bot cannot silently reset its model.
        if (current) return current;
        if (response.current && nextModels.some(item => item.name === response.current)) return response.current;
        return '';
      });
    } catch (e: any) {
      setModels([]);
      setModelError(e?.message || String(e));
    } finally {
      setModelsLoading(false);
    }
  }, [agentType]);

  useEffect(() => {
    if (step === 3) void loadModels();
  }, [step, loadModels]);

  const browseDirectory = async () => {
    setDirectoryPicking(true);
    setError('');
    try {
      const result = await selectSetupDirectory();
      if (!result.cancelled && result.path) setWorkDir(result.path);
    } catch (e: any) {
      setError(e?.message || String(e));
    } finally {
      setDirectoryPicking(false);
    }
  };

  const modelOptions = useMemo(() => {
    if (!model || models.some(item => item.name === model)) return models;
    return [{ name: model }, ...models];
  }, [model, models]);

  const next = () => { setError(''); setStep(value => Math.min(5, value + 1)); };
  const back = () => { setError(''); setStep(value => Math.max(1, value - 1)); };

  const choosePlatform = (platform: SetupPlatform) => {
    setPlatformType(platform.key);
    setOptions(platform.key === 'cloud_web' ? { transport: 'websocket' } : {});
    setShowAdvancedFields(false);
  };

  const requiredComplete = editID && platformType === originalPlatformType
    ? !!selectedPlatform
    : selectedPlatform?.fields.every(field => !field.required || !fieldVisible(field, options) || String(options[field.key] ?? '').trim()) ?? false;

  const save = async () => {
    setSaving(true);
    setError('');
    try {
      const request = {
        display_name: displayName.trim(),
        agent_type: agentType,
        work_dir: workDir.trim(),
        permission_mode: permissionMode,
        model: model.trim() || undefined,
        reasoning_effort: reasoningEffort || undefined,
        reply_footer: replyFooter,
        thinking_messages: thinkingMessages,
        tool_messages: toolMessages,
        platform_type: platformType,
        options,
      };
      if (editID) await updateBot(editID, request);
      else await createBot(request);
      await waitUntilReady();
      navigate('/', { replace: true });
    } catch (e: any) {
      setError(e?.message || String(e));
    } finally {
      setSaving(false);
    }
  };

  if (!catalog && !error) return <WizardShell step={step}><div className="py-20 flex justify-center"><Loader2 className="animate-spin text-accent" /></div></WizardShell>;

  return (
    <WizardShell step={step}>
      {error && <div className="mb-4 rounded-xl bg-red-50 dark:bg-red-950/30 text-red-600 p-3 text-sm">{error}</div>}

      {step === 1 && (
        <div className="space-y-5">
          <StepTitle title={t('simple.wizard.detectTitle')} description={t('simple.wizard.detectDescription')} />
          <div className="grid sm:grid-cols-2 gap-3">
            {agents.map(agent => {
              const status = health.find(item => item.key === agent.key);
              const available = !status || (status.installed && status.logged_in);
              return (
                <button key={agent.key} onClick={() => setAgentType(agent.key)} className={`p-4 rounded-xl border text-left transition ${agentType === agent.key ? 'border-accent bg-accent/5 ring-1 ring-accent/30' : 'border-gray-200 dark:border-gray-700'}`}>
                  <div className="flex items-center justify-between"><strong className="text-gray-900 dark:text-white">{agent.label}</strong>{available ? <CheckCircle2 size={17} className="text-emerald-500" /> : <XCircle size={17} className="text-red-500" />}</div>
                  <p className="text-xs text-gray-400 mt-2">{status?.version || (status ? status.problem : t('simple.wizard.advancedAgent'))}</p>
                  {status?.guide && !available && <p className="text-xs text-red-500 mt-2">{status.guide}</p>}
                </button>
              );
            })}
          </div>
          <button className="text-xs text-gray-500 flex items-center gap-1" onClick={() => setAdvancedAgents(value => !value)}><ChevronDown size={13} className={advancedAgents ? 'rotate-180' : ''} />{advancedAgents ? t('simple.wizard.hideAdvanced') : t('simple.wizard.showAdvanced')}</button>
          <WizardActions next={next} nextDisabled={agentBlocked || !agentType} />
        </div>
      )}

      {step === 2 && (
        <div className="space-y-5">
          <StepTitle title={t('simple.wizard.nameTitle')} description={t('simple.wizard.nameDescription')} />
          <Input label={t('simple.wizard.botName')} value={displayName} onChange={event => setDisplayName(event.target.value)} placeholder={t('simple.wizard.botNamePlaceholder')} autoFocus />
          <WizardActions back={back} next={next} nextDisabled={!displayName.trim()} />
        </div>
      )}

      {step === 3 && (
        <div className="space-y-4">
          <StepTitle title={t('simple.wizard.agentTitle')} description={t('simple.wizard.agentDescription')} />
          <div className="grid grid-cols-[minmax(0,1fr)_auto] gap-2 items-end">
            <Input label={t('setup.workDir', 'Working directory')} value={workDir} onChange={event => setWorkDir(event.target.value)} placeholder="D:\\projects\\my-app" />
            <Button type="button" variant="secondary" onClick={browseDirectory} loading={directoryPicking}><FolderOpen size={15} />{t('simple.wizard.browseDirectory')}</Button>
          </div>
          <div className="grid sm:grid-cols-2 gap-3">
            <Select label={t('simple.wizard.permissionMode')} value={permissionMode} onChange={setPermissionMode} options={selectedAgent?.modes || ['default']} />
            <Select label={t('simple.wizard.reasoningEffort')} value={reasoningEffort} onChange={setReasoningEffort} options={['low', 'medium', 'high', 'xhigh']} />
          </div>
          <div className="space-y-1.5">
            <div className="flex items-center justify-between gap-2">
              <label htmlFor="setup-model" className="text-sm font-medium text-gray-700 dark:text-gray-300">{t('simple.wizard.model')}</label>
              <Button type="button" variant="ghost" size="sm" onClick={() => void loadModels()} disabled={modelsLoading} aria-label={t('simple.wizard.refreshModels')}><RefreshCw size={14} className={modelsLoading ? 'animate-spin' : ''} />{t('simple.wizard.refreshModels')}</Button>
            </div>
            <select id="setup-model" value={model} onChange={event => setModel(event.target.value)} disabled={modelsLoading && modelOptions.length === 0} className="w-full px-3 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-900 dark:text-white disabled:opacity-60">
              <option value="">{modelsLoading ? t('simple.wizard.loadingModels') : t('simple.wizard.useAgentDefault')}</option>
              {modelOptions.map(option => <option key={option.name} value={option.name}>{option.description ? `${option.name} — ${option.description}` : option.name}</option>)}
            </select>
            {modelError && <p className="text-xs text-red-500">{t('simple.wizard.modelLoadFailed')}: {modelError}</p>}
          </div>
          <div className="flex items-start gap-2 rounded-xl bg-emerald-50 dark:bg-emerald-950/20 p-3 text-xs text-emerald-700 dark:text-emerald-300"><ShieldCheck size={16} className="shrink-0" />{t('simple.wizard.safeModeHint')}</div>
          <label className="flex items-center justify-between gap-4 rounded-xl border border-gray-200 dark:border-gray-700 p-3 cursor-pointer">
            <span>
              <span className="block text-sm font-medium text-gray-800 dark:text-gray-200">{t('simple.wizard.showReplyFooter')}</span>
              <span className="block text-xs text-gray-500 mt-0.5">{t('simple.wizard.showReplyFooterHint')}</span>
            </span>
            <input aria-label={t('simple.wizard.showReplyFooter')} type="checkbox" checked={replyFooter} onChange={event => setReplyFooter(event.target.checked)} className="h-4 w-4 accent-emerald-600" />
          </label>
          <label className="flex items-center justify-between gap-4 rounded-xl border border-gray-200 dark:border-gray-700 p-3 cursor-pointer">
            <span>
              <span className="block text-sm font-medium text-gray-800 dark:text-gray-200">{t('settings.thinkingMessages', 'Thinking messages')}</span>
              <span className="block text-xs text-gray-500 mt-0.5">{t('settings.thinkingMessagesHint', 'Show or hide intermediate thinking messages')}</span>
            </span>
            <input aria-label={t('settings.thinkingMessages', 'Thinking messages')} type="checkbox" checked={thinkingMessages} onChange={event => setThinkingMessages(event.target.checked)} className="h-4 w-4 accent-emerald-600" />
          </label>
          <label className="flex items-center justify-between gap-4 rounded-xl border border-gray-200 dark:border-gray-700 p-3 cursor-pointer">
            <span>
              <span className="block text-sm font-medium text-gray-800 dark:text-gray-200">{t('settings.toolMessages', 'Tool progress')}</span>
              <span className="block text-xs text-gray-500 mt-0.5">{t('settings.toolMessagesHint', 'Show or hide tool calls and results')}</span>
            </span>
            <input aria-label={t('settings.toolMessages', 'Tool progress')} type="checkbox" checked={toolMessages} onChange={event => setToolMessages(event.target.checked)} className="h-4 w-4 accent-emerald-600" />
          </label>
          <WizardActions back={back} next={next} nextDisabled={!workDir.trim()} />
        </div>
      )}

      {step === 4 && (
        <div className="space-y-4">
          <StepTitle title={t('simple.wizard.platformTitle')} description={t('simple.wizard.platformDescription')} />
          {!selectedPlatform ? (
            <div className="grid sm:grid-cols-2 gap-2 max-h-[430px] overflow-y-auto pr-1">
              {catalog?.platforms.map(platform => <button key={platform.key} onClick={() => choosePlatform(platform)} className="p-3 rounded-xl border border-gray-200 dark:border-gray-700 text-left hover:border-accent/60"><div className="font-medium text-sm text-gray-900 dark:text-white">{platform.label}</div><div className="text-xs text-gray-400 mt-1">{platform.qr ? t('simple.wizard.qrSetup') : t('simple.wizard.formSetup')}</div></button>)}
            </div>
          ) : selectedPlatform.qr ? (
            <QRSetup platform={selectedPlatform} onAuthorized={(actualPlatform, values) => { setPlatformType(actualPlatform); setOptions(values); }} />
          ) : (
            <GeneratedPlatformForm platform={selectedPlatform} values={options} onChange={setOptions} showAdvanced={showAdvancedFields} setShowAdvanced={setShowAdvancedFields} />
          )}
          <div className="flex justify-between gap-2 pt-2">
            <Button variant="secondary" onClick={selectedPlatform ? () => { setPlatformType(''); setOptions({}); } : back}><ChevronLeft size={14} /> {t('common.back')}</Button>
            <Button onClick={next} disabled={!selectedPlatform || !requiredComplete}><span>{t('setup.next', 'Next')}</span><ChevronRight size={14} /></Button>
          </div>
        </div>
      )}

      {step === 5 && (
        <div className="space-y-5">
          <StepTitle title={t('simple.wizard.finishTitle')} description={t('simple.wizard.finishDescription')} />
          <div className="rounded-xl border border-gray-200 dark:border-gray-700 divide-y divide-gray-100 dark:divide-gray-800 text-sm">
            <Review label={t('simple.wizard.botName')} value={displayName} />
            <Review label="Agent" value={agentType} />
            <Review label={t('setup.workDir', 'Working directory')} value={workDir} />
            <Review label={t('simple.wizard.platform')} value={selectedPlatform?.label || platformType} />
          </div>
          <p className="text-xs text-gray-500">{t('simple.wizard.restartNotice')}</p>
          <WizardActions back={back} next={save} nextLabel={t('simple.wizard.saveStart')} loading={saving} />
        </div>
      )}
    </WizardShell>
  );
}

function WizardShell({ step, children }: { step: number; children: React.ReactNode }) {
  return <div className="min-h-screen bg-gray-50 dark:bg-black flex items-center justify-center p-4"><div className="w-full max-w-2xl"><div className="flex items-center justify-center gap-2 mb-6"><div className="w-9 h-9 rounded-xl bg-accent text-white flex items-center justify-center"><Bot size={19} /></div><span className="font-bold text-lg text-gray-900 dark:text-white">CC<span className="text-accent">-</span>Connect</span></div><div className="flex gap-1.5 mb-3">{[1, 2, 3, 4, 5].map(value => <div key={value} className={`h-1.5 flex-1 rounded-full ${value <= step ? 'bg-accent' : 'bg-gray-200 dark:bg-gray-800'}`} />)}</div><Card className="p-6 sm:p-8">{children}</Card></div></div>;
}

function StepTitle({ title, description }: { title: string; description: string }) { return <div><h1 className="text-xl font-bold text-gray-900 dark:text-white">{title}</h1><p className="text-sm text-gray-500 mt-1">{description}</p></div>; }
function WizardActions({ back, next, nextDisabled, nextLabel, loading }: { back?: () => void; next: () => void; nextDisabled?: boolean; nextLabel?: string; loading?: boolean }) { const { t } = useTranslation(); return <div className="flex justify-between pt-3">{back ? <Button variant="secondary" onClick={back}><ChevronLeft size={14} />{t('common.back')}</Button> : <span />}<Button onClick={next} disabled={nextDisabled} loading={loading}>{nextLabel || t('setup.next', 'Next')} {!loading && <ChevronRight size={14} />}</Button></div>; }
function Select({ label, value, onChange, options }: { label: string; value: string; onChange: (value: string) => void; options: string[] }) { return <label className="text-xs font-medium text-gray-600 dark:text-gray-400">{label}<select value={value} onChange={event => onChange(event.target.value)} className="mt-1 w-full px-3 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-900 dark:text-white">{options.map(option => <option key={option}>{option}</option>)}</select></label>; }
function Review({ label, value }: { label: string; value: string }) { return <div className="flex gap-4 justify-between p-3"><span className="text-gray-500">{label}</span><span className="text-gray-900 dark:text-white text-right truncate">{value}</span></div>; }

function fieldVisible(field: SetupField, values: Values) { if (!field.show_when) return true; return Object.entries(field.show_when).every(([key, allowed]) => allowed.includes(String(values[key] ?? ''))); }

function GeneratedPlatformForm({ platform, values, onChange, showAdvanced, setShowAdvanced }: { platform: SetupPlatform; values: Values; onChange: (values: Values) => void; showAdvanced: boolean; setShowAdvanced: (value: boolean) => void }) {
  const { t } = useTranslation();
  const fields = platform.fields.filter(field => fieldVisible(field, values));
  const visible = fields.filter(field => field.group !== 'advanced' || showAdvanced);
  const advancedCount = fields.filter(field => field.group === 'advanced').length;
  const set = (key: string, value: any) => onChange({ ...values, [key]: value });
  return <div className="space-y-3"><h3 className="font-semibold text-gray-900 dark:text-white">{platform.label}</h3>{visible.map(field => <GeneratedField key={field.key} field={field} value={values[field.key]} onChange={value => set(field.key, value)} />)}{advancedCount > 0 && <button className="text-xs text-gray-500 flex items-center gap-1" onClick={() => setShowAdvanced(!showAdvanced)}><ChevronDown size={13} />{t('setup.advancedOptions', 'Advanced options')} ({advancedCount})</button>}</div>;
}

function GeneratedField({ field, value, onChange }: { field: SetupField; value: any; onChange: (value: any) => void }) {
  const { t } = useTranslation();
  const label = t(field.label, field.label);
  if (field.type === 'boolean') return <label className="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300"><input type="checkbox" checked={!!value} onChange={event => onChange(event.target.checked)} />{label}</label>;
  if (field.type === 'select') return <Select label={label} value={value || field.options?.[0] || ''} onChange={onChange} options={field.options || []} />;
  return <Input label={`${label}${field.required ? ' *' : ''}`} type={field.secret ? 'password' : field.type === 'number' ? 'number' : 'text'} value={value ?? ''} onChange={event => onChange(field.type === 'number' ? Number(event.target.value) : event.target.value)} placeholder={field.placeholder} />;
}

function QRSetup({ platform, onAuthorized }: { platform: SetupPlatform; onAuthorized: (platform: string, values: Values) => void }) {
  const { t } = useTranslation();
  const [qr, setQR] = useState('');
  const [state, setState] = useState<'loading' | 'waiting' | 'done' | 'error'>('loading');
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;
    const sleep = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));
    const run = async () => {
      try {
        if (platform.key === 'weixin') {
          const begin = await setupWeixinBegin();
          if (cancelled) return;
          setQR(begin.qr_url); setState('waiting');
          while (!cancelled) {
            const poll = await setupWeixinPoll(begin.qr_key);
            if (poll.status === 'confirmed') {
              onAuthorized('weixin', { token: poll.bot_token, base_url: poll.base_url, ilink_bot_id: poll.ilink_bot_id, ilink_user_id: poll.ilink_user_id });
              setState('done'); return;
            }
            if (poll.status === 'expired') throw new Error(t('setup.expired', 'QR code expired'));
            await sleep(600);
          }
        } else {
          const begin = await setupFeishuBegin();
          if (cancelled) return;
          setQR(begin.qr_url); setState('waiting');
          let interval = begin.interval || 5;
          let baseUrl = '';
          while (!cancelled) {
            const poll = await setupFeishuPoll(begin.device_code, baseUrl || undefined);
            if (poll.base_url) baseUrl = poll.base_url;
            if (poll.slow_down) interval += 5;
            if (poll.status === 'completed') {
              onAuthorized(poll.platform || platform.key, { app_id: poll.app_id, app_secret: poll.app_secret, owner_open_id: poll.owner_open_id });
              setState('done'); return;
            }
            if (poll.status === 'denied' || poll.status === 'expired' || poll.status === 'error') throw new Error(poll.error || poll.status);
            await sleep(interval * 1000);
          }
        }
      } catch (e: any) {
        if (!cancelled) { setError(e?.message || String(e)); setState('error'); }
      }
    };
    run();
    return () => { cancelled = true; };
  }, [platform.key, t]);

  return <div className="py-4 flex flex-col items-center gap-3">{state === 'loading' && <Loader2 className="animate-spin text-accent" />}{qr && <div className="bg-white p-3 rounded-xl border"><QRCodeSVG value={qr} size={190} /></div>}{state === 'waiting' && <p className="text-sm text-gray-500">{t('simple.wizard.scanQR')}</p>}{state === 'done' && <p className="text-sm text-emerald-600 flex items-center gap-2"><Check size={16} />{t('simple.wizard.authorized')}</p>}{state === 'error' && <p className="text-sm text-red-500">{error}</p>}</div>;
}
