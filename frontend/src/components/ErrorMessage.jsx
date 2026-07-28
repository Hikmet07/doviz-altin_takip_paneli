function ErrorMessage({ message = "Bir hata oluştu." }) {
  return (
    <div className="flex items-start gap-3 rounded-2xl border border-rose-200 bg-rose-50/80 p-5 text-rose-800 shadow-xs">
      <span className="text-xl">⚠️</span>
      <div>
        <h4 className="font-bold text-sm">Bağlantı Uyarısı</h4>
        <p className="text-xs mt-0.5 text-rose-700 leading-relaxed">{message}</p>
      </div>
    </div>
  );
}

export default ErrorMessage;