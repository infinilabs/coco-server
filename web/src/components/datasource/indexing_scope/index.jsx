import { Switch } from "antd";
import { useState } from "react";

export const IndexingScope = ({
  value = {
    indexing_books: true,
    indexing_docs: true,
    indexing_groups: true,
    indexing_users: false,
    include_private_book: true,
  },
  onChange,
})=>{
  const [innerV, setValue] = useState(value);
  const onPartChange = (key, v)=>{
    if(innerV[key]){
      setValue(oldV=>{
        const newV =  {
          ...oldV,
          key: v,
        }
        if(typeof onChange === "function"){
          onChange(newV);
        }
        return newV;
      })
    }
  }
  return <div>
    <div className="flex justify-between max-w-200px pb-10px pt-5px"><span className="text-gray-400">Indexing Books</span>
      <Switch onChange={(v)=>{
        onPartChange("indexing_books", v)
      }} checked={innerV.indexing_books}/>
    </div>
    <div className="flex justify-between max-w-200px py-10px"><span className="text-gray-400">Indexing Docs</span>
      <Switch onChange={(v)=>{
        onPartChange("indexing_docs", v)
      }} checked={innerV.indexing_docs}/>
    </div>
    <div className="flex justify-between max-w-200px py-10px"><span className="text-gray-400">Indexing Groups</span> 
      <Switch onChange={(v)=>{
        onPartChange("indexing_groups", v)
      }} checked={innerV.indexing_groups}/>
    </div>
    <div className="flex justify-between max-w-200px py-10px"><span className="text-gray-400">Indexing Users</span> 
      <Switch onChange={(v)=>{
        onPartChange("indexing_users", v)
      }} checked={innerV.indexing_users}/>
      </div>
    <div className="flex justify-between max-w-200px py-10px"><span className="text-gray-400">Icnlude Private Book</span>
     <Switch onChange={(v)=>{
        onPartChange("include_private_book", v)
      }} checked={innerV.include_private_book}/>
      </div>
  </div>
}