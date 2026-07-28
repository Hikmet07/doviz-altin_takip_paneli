function Loading({ text = "Veriler yükleniyor..." }) {
  return (
    <div className="flex h-64 flex-col items-center justify-center rounded-2xl border border-slate-200 bg-white p-8 text-center shadow-xs">
      <div className="h-10 w-10 animate-spin rounded-full border-4 border-indigo-200 border-t-indigo-600"></div>
      <p className="mt-4 text-sm font-semibold text-slate-700">{text}</p>
    </div>
  );
}

export default Loading;