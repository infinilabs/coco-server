import UserForm from './UserForm';

const Guide = memo(() => {
  return (
    <div className="items-left size-full flex flex-col justify-center overflow-auto px-10%">
      <div className="w-440px">
        <UserForm />
      </div>
    </div>
  );
});

export default Guide;
