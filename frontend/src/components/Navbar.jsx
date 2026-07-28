import { NavLink } from "react-router-dom";

function Navbar() {
  const getLinkClass = ({ isActive }) =>
    isActive
      ? "rounded-xl bg-slate-900 px-4 py-2 text-sm font-semibold text-white shadow-xs transition-all"
      : "rounded-xl px-4 py-2 text-sm font-medium text-slate-600 hover:bg-slate-100 hover:text-slate-900 transition-all";

  return (
    <header className="sticky top-0 z-50 border-b border-slate-200/80 bg-white/90 backdrop-blur-md shadow-xs">
      <div className="mx-auto flex max-w-7xl flex-col gap-4 px-4 py-3.5 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-indigo-600 font-bold text-white shadow-sm">
            📈
          </div>
          <div>
            <h1 className="text-lg font-bold text-slate-900 tracking-tight leading-none">
              Altın & Döviz Takip Paneli
            </h1>
            <p className="text-xs text-slate-500 mt-1 font-medium">
              Altınkaynak Canlı Piyasa Verileri
            </p>
          </div>
        </div>

        <nav className="flex items-center gap-1">
          <NavLink to="/dashboard" className={getLinkClass}>
            Dashboard
          </NavLink>

          <NavLink to="/markets" className={getLinkClass}>
            Piyasalar
          </NavLink>

          <NavLink to="/history" className={getLinkClass}>
            Grafikler
          </NavLink>
        </nav>
      </div>
    </header>
  );
}

export default Navbar;