interface Props extends React.ComponentProps<'div'> {}

const ListContainer: FC<Props> = memo(({ children }) => {
  return (
    <div className="min-h-full flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
      {children}
    </div>
  );
});

export default ListContainer;
