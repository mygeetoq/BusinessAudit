import { Landmark, Upload } from "lucide-react";
import { useEffect, useState } from "react";
import { api, ApiError } from "../api/client";
import { ErrorNote, Spinner } from "../components/Bits";

type Account = { bank_name: string; account_id: string; mode: "manual_csv" };
type Report = { imported: number; skipped: number; days_updated: number; errors: string[] };

export default function BankAccount() {
  const [account, setAccount] = useState<Account | null>(null);
  const [bankName, setBankName] = useState("");
  const [accountID, setAccountID] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  useEffect(() => {
    api<Account>("/api/my/bank-account").then((value) => {
      setAccount(value);
      setBankName(value.bank_name);
      setAccountID(value.account_id);
    }).catch((err) => setError(err instanceof ApiError ? err.message : "Не удалось загрузить счёт"));
  }, []);

  async function connect(e: React.FormEvent) {
    e.preventDefault(); setBusy(true); setError(""); setMessage("");
    try {
      const value = await api<Account>("/api/my/bank-account", {
        method: "PUT", body: JSON.stringify({ bank_name: bankName, account_id: accountID }),
      });
      setAccount(value);
      setMessage("Реквизиты счёта сохранены.");
    } catch (err) { setError(err instanceof ApiError ? err.message : "Не удалось сохранить счёт"); }
    finally { setBusy(false); }
  }

  async function upload(e: React.FormEvent) {
    e.preventDefault();
    if (!file) return;
    setBusy(true); setError(""); setMessage("");
    try {
      const form = new FormData(); form.append("file", file);
      const report = await api<Report>("/api/my/bank-statement", { method: "POST", body: form });
      setMessage(`Загружено операций: ${report.imported}; обновлено дней: ${report.days_updated}.`);
      setFile(null);
    } catch (err) { setError(err instanceof ApiError ? err.message : "Не удалось загрузить выписку"); }
    finally { setBusy(false); }
  }

  if (!account && !error) return <Spinner />;
  return <div className="flex flex-col gap-4">
    <div className="card flex items-center gap-3 p-5">
      <Landmark className="size-7 text-brand" />
      <div><h1 className="text-lg font-bold">Данные бизнеса</h1>
        <p className="text-sm text-ink-2">Загрузите операции из CSV. Банк и номер счёта можно указать отдельно.</p></div>
    </div>
    {error && <ErrorNote message={error} />}
    {message && <div className="rounded-xl bg-good-soft p-3 text-sm text-good">{message}</div>}
    <div className="grid gap-4 md:grid-cols-2">
      <form onSubmit={connect} className="card flex flex-col gap-3 p-5">
        <h2 className="font-semibold">Банк и счёт · необязательно для CSV</h2>
        <label className="text-sm">Название банка
          <input className="input mt-1" value={bankName} onChange={(e) => setBankName(e.target.value)} minLength={2} maxLength={120} required placeholder="Ваш банк" />
        </label>
        <label className="text-sm">Номер или идентификатор бизнес-счёта
          <input className="input mt-1" value={accountID} onChange={(e) => setAccountID(e.target.value)} minLength={5} maxLength={50} required placeholder="40702…" />
        </label>
        <p className="text-xs text-ink-3">Реквизиты нужны для будущего подключения банковского API. Сохранение номера не подтверждает владение счётом. Уже указанный счёт нельзя заменить после появления операций.</p>
        <button className="btn-primary self-start" disabled={busy}>Сохранить счёт</button>
      </form>
      <form onSubmit={upload} className="card flex flex-col gap-3 p-5">
        <h2 className="font-semibold">Загрузите операции из CSV</h2>
        <p className="text-sm text-ink-2">Номер счёта вводить не нужно: операции попадут в ваш профиль.</p>
        <p className="text-sm text-ink-2">Экспортируйте операции из интернет-банка и подготовьте CSV с тремя колонками: <code>date,amount,type</code>. Подходят также <code>дата,сумма,тип</code>, разделитель — запятая или точка с запятой.</p>
        <p className="text-sm text-ink-2"><code>2026-10-01,1500.00,income</code><br /><code>2026-10-02,200.00,return</code></p>
        <p className="text-xs text-ink-3">Здесь учитываются продажи и возвраты. Остальные списания добавляйте в разделе «Расходы». Для автоматической синхронизации нужен API-шлюз вашего банка.</p>
        <label className="text-sm">CSV-файл до 10 МБ
          <input className="input mt-1" type="file" accept=".csv,text/csv" onChange={(e) => setFile(e.target.files?.[0] ?? null)} required />
        </label>
        <button className="btn-primary self-start" disabled={busy || !file}><Upload className="size-4" /> Загрузить</button>
      </form>
    </div>
  </div>;
}
