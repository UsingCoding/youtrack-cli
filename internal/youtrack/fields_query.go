package youtrack

const issueFields = "id,idReadable,summary,description,created,updated,resolved,project(id,name,shortName),reporter(id,login,fullName),updater(id,login,fullName),tags(id,name),customFields(id,name,$type,value(id,name,login,fullName,minutes,presentation,text,markdownText,isResolved),possibleEvents(id,presentation))"
const projectFields = "id,name,shortName"
const projectCustomFieldFields = "id,$type,canBeEmpty,field(id,name,fieldType(id,valueType,isMultiValue)),bundle(id)"
const bundleValueFields = "id,name,archived"
const userFields = "id,login,fullName"
const tagFields = "id,name"
const groupFields = "id,name"
