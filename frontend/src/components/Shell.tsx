import { NavLink, Outlet, useLocation, useSearchParams } from "react-router-dom";
import { fromInput, toInput } from "../api";

export default function Shell() {
  const location = useLocation();
  const hideWindow = location.pathname === "/import";
  const [params, setParams] = useSearchParams();
  const from = toInput(params.get("from"));
  const to = toInput(params.get("to"));

  const setWindow = (nextFrom: string, nextTo: string) => {
    const next = new URLSearchParams(params);
    if (nextFrom) next.set("from", fromInput(nextFrom));
    else next.delete("from");
    if (nextTo) next.set("to", fromInput(nextTo));
    else next.delete("to");
    setParams(next, { replace: true });
  };

  return (
    <div className="shell">
      <header className="topbar">
        <div className="brand">
          <small>RED / UTC</small>
          <strong>调用质量台</strong>
        </div>
        <nav className="nav">
          <NavLink to={{ pathname: "/", search: location.search }} end>看板</NavLink>
          <NavLink to="/import">导入</NavLink>
          <NavLink to={{ pathname: "/endpoints", search: location.search }}>聚合</NavLink>
          <NavLink to={{ pathname: "/requests", search: location.search }}>原始记录</NavLink>
        </nav>
        {!hideWindow && (
          <div className="window">
            <label>
              开始 (UTC)
              <input type="datetime-local" value={from} onChange={(e) => setWindow(e.target.value, to)} />
            </label>
            <label>
              结束 (UTC)
              <input type="datetime-local" value={to} onChange={(e) => setWindow(from, e.target.value)} />
            </label>
          </div>
        )}
      </header>
      <Outlet />
    </div>
  );
}
