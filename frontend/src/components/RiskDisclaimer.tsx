export default function RiskDisclaimer({
  className = "",
}: {
  className?: string;
}) {
  return (
    <div
      role="note"
      className={`rounded-xl border border-gold-500/30 bg-gold-500/[0.07] p-4 text-sm text-cream/80 ${className}`}
    >
      <p className="font-semibold text-gold-300">Lưu ý khi đầu tư</p>
      <p className="mt-1 leading-relaxed">
        Đầu tư cổ phần là hình thức đồng hành và chia sẻ thành quả cùng doanh
        nghiệp. Giá trị cổ phần và cổ tức gắn với kết quả kinh doanh thực tế nên
        có thể thay đổi theo thời gian. Nền tảng{" "}
        <strong className="text-cream">không cam kết một mức lợi nhuận cố định</strong>;
        quyền lợi nhận lại (nếu có) đến từ{" "}
        <strong className="text-cream">cổ tức do công ty công bố và chi trả</strong>.
        Vui lòng cân nhắc kỹ và đầu tư trong khả năng tài chính của mình.
      </p>
    </div>
  );
}
