interface Props extends React.ComponentProps<'div'> {}

const ListContainer: FC<Props> = memo(({ children, className = '' }) => {
  return (
    <div className={`min-h-full flex-col-stretch overflow-hidden lt-sm:overflow-auto ${className}`}>
      {children}
    </div>
  );
});

export default ListContainer;
